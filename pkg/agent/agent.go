package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/netobserv/gopipes/pkg/node"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/ebpf"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/exporter"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/flow"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/ifaces"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/metrics"
	promo "github.com/netobserv/netobserv-ebpf-agent/pkg/prometheus"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/gavv/monotime"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/sirupsen/logrus"
)

var alog = logrus.WithField("component", "agent.Flows")
var plog = logrus.WithField("component", "agent.Packets")

// Status of the agent service. Helps on the health report as well as making some asynchronous
// tests waiting for the agent to accept flows.
type Status int

const (
	StatusNotStarted Status = iota
	StatusStarting
	StatusStarted
	StatusStopping
	StatusStopped
)

func (s Status) String() string {
	switch s {
	case StatusNotStarted:
		return "StatusNotStarted"
	case StatusStarting:
		return "StatusStarting"
	case StatusStarted:
		return "StatusStarted"
	case StatusStopping:
		return "StatusStopping"
	case StatusStopped:
		return "StatusStopped"
	default:
		return "invalid"
	}
}

func configureInformer(cfg *Config, log *logrus.Entry) ifaces.Informer {
	var informer ifaces.Informer
	switch cfg.ListenInterfaces {
	case ListenPoll:
		log.WithField("period", cfg.ListenPollPeriod).
			Debug("listening for new interfaces: use polling")
		informer = ifaces.NewPoller(cfg.ListenPollPeriod, cfg.BuffersLength)
	case ListenWatch:
		log.Debug("listening for new interfaces: use watching")
		informer = ifaces.NewWatcher(cfg.BuffersLength)
	default:
		log.WithField("providedValue", cfg.ListenInterfaces).
			Warn("wrong interface listen method. Using file watcher as default")
		informer = ifaces.NewWatcher(cfg.BuffersLength)
	}
	return informer

}

func interfaceListener(ctx context.Context, ifaceEvents <-chan ifaces.Event, slog *logrus.Entry, eventAdded func(iface ifaces.Interface)) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("stopping interfaces' listener")
			return
		case event := <-ifaceEvents:
			slog.WithField("event", event).Debug("received event")
			switch event.Type {
			case ifaces.EventAdded:
				eventAdded(event.Interface)
			case ifaces.EventDeleted:
				// qdiscs, ingress and egress filters are automatically deleted so we don't need to
				// specifically detach them from the ebpfFetcher
			default:
				slog.WithField("event", event).Warn("unknown event type")
			}
		}
	}
}

// Flows reporting agent
type Flows struct {
	cfg *Config

	// input data providers
	interfaces ifaces.Informer
	filter     InterfaceFilter
	ebpf       ebpfFlowFetcher

	// processing nodes to be wired in the buildAndStartPipeline method
	mapTracer *flow.MapTracer
	rbTracer  *flow.RingBufTracer
	accounter *flow.Accounter
	limiter   *flow.CapacityLimiter
	deduper   node.MiddleFunc[[]*flow.Record, []*flow.Record]
	exporter  node.TerminalFunc[[]*flow.Record]

	// elements used to decorate flows with extra information
	interfaceNamer flow.InterfaceNamer
	agentIP        net.IP

	status      Status
	promoServer *http.Server
}

// ebpfFlowFetcher abstracts the interface of ebpf.FlowFetcher to allow dependency injection in tests
type ebpfFlowFetcher interface {
	io.Closer
	Register(iface ifaces.Interface) error
	AttachTCX(iface ifaces.Interface) error

	LookupAndDeleteMap(*metrics.Metrics) map[ebpf.BpfFlowId][]ebpf.BpfFlowMetrics
	DeleteMapsStaleEntries(timeOut time.Duration)
	ReadRingBuf() (ringbuf.Record, error)
}

// FlowsAgent instantiates a new agent, given a configuration.
func FlowsAgent(cfg *Config) (*Flows, error) {
	alog.Info("initializing Flows agent")

	// manage deprecated configs
	manageDeprecatedConfigs(cfg)

	// configure informer for new interfaces
	var informer = configureInformer(cfg, alog)

	alog.Debug("acquiring Agent IP")
	agentIP, err := fetchAgentIP(cfg)
	if err != nil {
		return nil, fmt.Errorf("acquiring Agent IP: %w", err)
	}
	alog.Debug("agent IP: " + agentIP.String())

	// initialize metrics
	metricsSettings := &metrics.Settings{
		PromConnectionInfo: metrics.PromConnectionInfo{
			Address: cfg.MetricsServerAddress,
			Port:    cfg.MetricsPort,
		},
		Prefix: cfg.MetricsPrefix,
	}
	if cfg.MetricsTLSCertPath != "" && cfg.MetricsTLSKeyPath != "" {
		metricsSettings.PromConnectionInfo.TLS = &metrics.PromTLS{
			CertPath: cfg.MetricsTLSCertPath,
			KeyPath:  cfg.MetricsTLSKeyPath,
		}
	}
	m := metrics.NewMetrics(metricsSettings)

	// configure selected exporter
	exportFunc, err := buildFlowExporter(cfg, m)
	if err != nil {
		return nil, err
	}

	ingress, egress := flowDirections(cfg)
	debug := false
	if cfg.LogLevel == logrus.TraceLevel.String() || cfg.LogLevel == logrus.DebugLevel.String() {
		debug = true
	}

	ebpfConfig := &ebpf.FlowFetcherConfig{
		EnableIngress:    ingress,
		EnableEgress:     egress,
		Debug:            debug,
		Sampling:         cfg.Sampling,
		CacheMaxSize:     cfg.CacheMaxFlows,
		PktDrops:         cfg.EnablePktDrops,
		DNSTracker:       cfg.EnableDNSTracking,
		EnableRTT:        cfg.EnableRTT,
		EnableFlowFilter: cfg.EnableFlowFilter,
		FilterConfig: &ebpf.FilterConfig{
			FilterAction:          cfg.FilterAction,
			FilterDirection:       cfg.FilterDirection,
			FilterIPCIDR:          cfg.FilterIPCIDR,
			FilterProtocol:        cfg.FilterProtocol,
			FilterPeerIP:          cfg.FilterPeerIP,
			FilterDestinationPort: ebpf.ConvertFilterPortsToInstr(cfg.FilterDestinationPort, cfg.FilterDestinationPortRange),
			FilterSourcePort:      ebpf.ConvertFilterPortsToInstr(cfg.FilterSourcePort, cfg.FilterSourcePortRange),
			FilterPort:            ebpf.ConvertFilterPortsToInstr(cfg.FilterPort, cfg.FilterPortRange),
		},
	}

	fetcher, err := ebpf.NewFlowFetcher(ebpfConfig)
	if err != nil {
		return nil, err
	}

	return flowsAgent(cfg, m, informer, fetcher, exportFunc, agentIP)
}

// flowsAgent is a private constructor with injectable dependencies, usable for tests
func flowsAgent(cfg *Config, m *metrics.Metrics,
	informer ifaces.Informer,
	fetcher ebpfFlowFetcher,
	exporter node.TerminalFunc[[]*flow.Record],
	agentIP net.IP,
) (*Flows, error) {
	var filter InterfaceFilter

	switch {
	case len(cfg.InterfaceIPs) > 0 && (len(cfg.Interfaces) > 0 || len(cfg.ExcludeInterfaces) > 0):
		return nil, fmt.Errorf("INTERFACES/EXCLUDE_INTERFACES and INTERFACE_IPS are mutually exclusive")

	case len(cfg.InterfaceIPs) > 0:
		// configure ip interface filter
		f, err := initIPInterfaceFilter(cfg.InterfaceIPs, IPsFromInterface)
		if err != nil {
			return nil, fmt.Errorf("configuring interface ip filter: %w", err)
		}
		filter = &f

	default:
		// configure allow/deny regexp interfaces filter
		f, err := initRegexpInterfaceFilter(cfg.Interfaces, cfg.ExcludeInterfaces)
		if err != nil {
			return nil, fmt.Errorf("configuring interface filters: %w", err)
		}
		filter = &f
	}

	registerer := ifaces.NewRegisterer(informer, cfg.BuffersLength)

	interfaceNamer := func(ifIndex int) string {
		iface, ok := registerer.IfaceNameForIndex(ifIndex)
		if !ok {
			return "unknown"
		}
		return iface
	}
	var promoServer *http.Server
	if cfg.MetricsEnable {
		promoServer = promo.InitializePrometheus(m.Settings)
	}

	samplingGauge := m.CreateSamplingRate()
	samplingGauge.Set(float64(cfg.Sampling))

	mapTracer := flow.NewMapTracer(fetcher, cfg.CacheActiveTimeout, cfg.StaleEntriesEvictTimeout, m)
	rbTracer := flow.NewRingBufTracer(fetcher, mapTracer, cfg.CacheActiveTimeout, m)
	accounter := flow.NewAccounter(cfg.CacheMaxFlows, cfg.CacheActiveTimeout, time.Now, monotime.Now, m)
	limiter := flow.NewCapacityLimiter(m)
	var deduper node.MiddleFunc[[]*flow.Record, []*flow.Record]
	if cfg.Deduper == DeduperFirstCome {
		deduper = flow.Dedupe(cfg.DeduperFCExpiry, cfg.DeduperJustMark, cfg.DeduperMerge, interfaceNamer, m)
	}

	return &Flows{
		ebpf:           fetcher,
		exporter:       exporter,
		interfaces:     registerer,
		filter:         filter,
		cfg:            cfg,
		mapTracer:      mapTracer,
		rbTracer:       rbTracer,
		accounter:      accounter,
		limiter:        limiter,
		deduper:        deduper,
		agentIP:        agentIP,
		interfaceNamer: interfaceNamer,
		promoServer:    promoServer,
	}, nil
}

func flowDirections(cfg *Config) (ingress, egress bool) {
	switch cfg.Direction {
	case DirectionIngress:
		return true, false
	case DirectionEgress:
		return false, true
	case DirectionBoth:
		return true, true
	default:
		alog.Warnf("unknown DIRECTION %q. Tracing both ingress and egress traffic", cfg.Direction)
		return true, true
	}
}

func buildFlowExporter(cfg *Config, m *metrics.Metrics) (node.TerminalFunc[[]*flow.Record], error) {
	switch cfg.Export {
	case "grpc":
		return buildGRPCExporter(cfg, m)
	case "kafka":
		return buildKafkaExporter(cfg, m)
	case "ipfix+udp":
		return buildIPFIXExporter(cfg, "udp")
	case "ipfix+tcp":
		return buildIPFIXExporter(cfg, "tcp")
	case "direct-flp":
		return buildFlowDirectFLPExporter(cfg)
	default:
		return nil, fmt.Errorf("wrong flow export type %s", cfg.Export)
	}
}

func buildGRPCExporter(cfg *Config, m *metrics.Metrics) (node.TerminalFunc[[]*flow.Record], error) {
	if cfg.TargetHost == "" || cfg.TargetPort == 0 {
		return nil, fmt.Errorf("missing target host or port: %s:%d",
			cfg.TargetHost, cfg.TargetPort)
	}
	grpcExporter, err := exporter.StartGRPCProto(cfg.TargetHost, cfg.TargetPort, cfg.GRPCMessageMaxFlows, m)
	if err != nil {
		return nil, err
	}
	return grpcExporter.ExportFlows, nil
}

func buildFlowDirectFLPExporter(cfg *Config) (node.TerminalFunc[[]*flow.Record], error) {
	flpExporter, err := exporter.StartDirectFLP(cfg.FLPConfig, cfg.BuffersLength)
	if err != nil {
		return nil, err
	}
	return flpExporter.ExportFlows, nil
}

func buildKafkaExporter(cfg *Config, m *metrics.Metrics) (node.TerminalFunc[[]*flow.Record], error) {
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("at least one Kafka broker is needed")
	}
	var compression compress.Compression
	if err := compression.UnmarshalText([]byte(cfg.KafkaCompression)); err != nil {
		return nil, fmt.Errorf("wrong Kafka compression value %s. Admitted values are "+
			"none, gzip, snappy, lz4, zstd: %w", cfg.KafkaCompression, err)
	}
	transport := kafkago.Transport{}
	if cfg.KafkaEnableTLS {
		tlsConfig, err := buildTLSConfig(cfg)
		if err != nil {
			return nil, err
		}
		transport.TLS = tlsConfig
	}
	if cfg.KafkaEnableSASL {
		mechanism, err := buildSASLConfig(cfg)
		if err != nil {
			return nil, err
		}
		transport.SASL = mechanism
	}
	return (&exporter.KafkaProto{
		Writer: &kafkago.Writer{
			Addr:      kafkago.TCP(cfg.KafkaBrokers...),
			Topic:     cfg.KafkaTopic,
			BatchSize: cfg.KafkaBatchMessages,
			// Assigning KafkaBatchSize to BatchBytes instead of BatchSize might be confusing here.
			// The reason is that the "standard" Kafka name for this variable is "batch.size",
			// which specifies the size of messages in terms of bytes, and not in terms of entries.
			// We have decided to hide this library implementation detail and expose to the
			// customer the common, standard name and meaning for batch.size
			BatchBytes: int64(cfg.KafkaBatchSize),
			// Segmentio's Kafka-go does not behave as standard Kafka library, and would
			// throttle any Write invocation until reaching the timeout.
			// Since we invoke write once each CacheActiveTimeout, we can safely disable this
			// timeout throttling
			// https://github.com/netobserv/flowlogs-pipeline/pull/233#discussion_r897830057
			BatchTimeout: time.Nanosecond,
			Async:        cfg.KafkaAsync,
			Compression:  compression,
			Transport:    &transport,
			Balancer:     &kafkago.Hash{},
		},
		Metrics: m,
	}).ExportFlows, nil
}

func buildIPFIXExporter(cfg *Config, proto string) (node.TerminalFunc[[]*flow.Record], error) {
	if cfg.TargetHost == "" || cfg.TargetPort == 0 {
		return nil, fmt.Errorf("missing target host or port: %s:%d",
			cfg.TargetHost, cfg.TargetPort)
	}
	ipfix, err := exporter.StartIPFIXExporter(cfg.TargetHost, cfg.TargetPort, proto)
	if err != nil {
		return nil, err
	}
	return ipfix.ExportFlows, nil
}

// Run a Flows agent. The function will keep running in the same thread
// until the passed context is canceled
func (f *Flows) Run(ctx context.Context) error {
	f.status = StatusStarting
	alog.Info("starting Flows agent")
	graph, err := f.buildAndStartPipeline(ctx)
	if err != nil {
		return fmt.Errorf("starting processing graph: %w", err)
	}

	f.status = StatusStarted
	alog.Info("Flows agent successfully started")
	<-ctx.Done()

	f.status = StatusStopping
	alog.Info("stopping Flows agent")
	if err := f.ebpf.Close(); err != nil {
		alog.WithError(err).Warn("eBPF resources not correctly closed")
	}

	alog.Debug("waiting for all nodes to finish their pending work")
	<-graph.Done()
	if f.promoServer != nil {
		alog.Debug("closing prometheus server")
		if err := f.promoServer.Close(); err != nil {
			alog.WithError(err).Warn("error when closing prometheus server")
		}
	}
	f.status = StatusStopped
	alog.Info("Flows agent stopped")
	return nil
}

func (f *Flows) Status() Status {
	return f.status
}

// interfacesManager uses an informer to check new/deleted network interfaces. For each running
// interface, it registers a flow ebpfFetcher that will forward new flows to the returned channel
// TODO: consider move this method and "onInterfaceAdded" to another type
func (f *Flows) interfacesManager(ctx context.Context) error {
	slog := alog.WithField("function", "interfacesManager")

	slog.Debug("subscribing for network interface events")
	ifaceEvents, err := f.interfaces.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("instantiating interfaces' informer: %w", err)
	}

	go interfaceListener(ctx, ifaceEvents, slog, f.onInterfaceAdded)

	return nil
}

// buildAndStartPipeline creates the ETL flow processing graph.
// For a more visual view, check the docs/architecture.md document.
func (f *Flows) buildAndStartPipeline(ctx context.Context) (*node.Terminal[[]*flow.Record], error) {

	alog.Debug("registering interfaces' listener in background")
	err := f.interfacesManager(ctx)
	if err != nil {
		return nil, err
	}

	alog.Debug("connecting flows' processing graph")
	mapTracer := node.AsStart(f.mapTracer.TraceLoop(ctx, f.cfg.ForceGC))
	rbTracer := node.AsStart(f.rbTracer.TraceLoop(ctx))

	accounter := node.AsMiddle(f.accounter.Account,
		node.ChannelBufferLen(f.cfg.BuffersLength))

	limiter := node.AsMiddle(f.limiter.Limit,
		node.ChannelBufferLen(f.cfg.BuffersLength))

	decorator := node.AsMiddle(flow.Decorate(f.agentIP, f.interfaceNamer),
		node.ChannelBufferLen(f.cfg.BuffersLength))

	ebl := f.cfg.ExporterBufferLength
	if ebl == 0 {
		ebl = f.cfg.BuffersLength
	}

	export := node.AsTerminal(f.exporter,
		node.ChannelBufferLen(ebl))

	rbTracer.SendsTo(accounter)

	if f.deduper != nil {
		deduper := node.AsMiddle(f.deduper, node.ChannelBufferLen(f.cfg.BuffersLength))
		mapTracer.SendsTo(deduper)
		accounter.SendsTo(deduper)
		deduper.SendsTo(limiter)
	} else {
		mapTracer.SendsTo(limiter)
		accounter.SendsTo(limiter)
	}
	limiter.SendsTo(decorator)
	decorator.SendsTo(export)

	alog.Debug("starting graph")
	mapTracer.Start()
	rbTracer.Start()
	return export, nil
}

func (f *Flows) onInterfaceAdded(iface ifaces.Interface) {
	// ignore interfaces that do not match the user configuration acceptance/exclusion lists
	allowed, err := f.filter.Allowed(iface.Name)
	if err != nil {
		alog.WithField("interface", iface).Errorf("encountered error determining if interface is allowed: %v", err)
		return
	}
	if !allowed {
		alog.WithField("interface", iface).
			Debug("interface does not match the allow/exclusion filters. Ignoring")
		return
	}
	alog.WithField("interface", iface).Info("interface detected. trying to attach TCX hook")
	if err := f.ebpf.AttachTCX(iface); err != nil {
		alog.WithField("interface", iface).WithError(err).
			Info("can't attach to TCx hook flow ebpfFetcher. fall back to use legacy TC hook")
		if err := f.ebpf.Register(iface); err != nil {
			alog.WithField("interface", iface).WithError(err).
				Warn("can't register flow ebpfFetcher. Ignoring")
			return
		}
	}
}
