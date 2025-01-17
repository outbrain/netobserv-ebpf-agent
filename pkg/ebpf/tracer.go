package ebpf

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/netobserv/netobserv-ebpf-agent/pkg/ifaces"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/kernel"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/metrics"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/gavv/monotime"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
)

// $BPF_CLANG and $BPF_CFLAGS are set by the Makefile.
//go:generate bpf2go -cc $BPF_CLANG -cflags $BPF_CFLAGS -target amd64,arm64,ppc64le,s390x -type flow_metrics_t -type flow_id_t -type flow_record_t -type pkt_drops_t -type dns_record_t -type global_counters_key_t -type direction_t -type filter_action_t Bpf ../../bpf/flows.c -- -I../../bpf/headers

const (
	qdiscType = "clsact"
	// ebpf map names as defined in bpf/maps_definition.h
	aggregatedFlowsMap = "aggregated_flows"
	dnsLatencyMap      = "dns_flows"
	// constants defined in flows.c as "volatile const"
	constSampling            = "sampling"
	constTraceMessages       = "trace_messages"
	constEnableRtt           = "enable_rtt"
	constEnableDNSTracking   = "enable_dns_tracking"
	constEnableFlowFiltering = "enable_flows_filtering"
	pktDropHook              = "kfree_skb"
	constPcaEnable           = "enable_pca"
	pcaRecordsMap            = "packet_record"
	tcEgressFilterName       = "tc/tc_egress_flow_parse"
	tcIngressFilterName      = "tc/tc_ingress_flow_parse"
)

var log = logrus.WithField("component", "ebpf.FlowFetcher")
var plog = logrus.WithField("component", "ebpf.PacketFetcher")

// FlowFetcher reads and forwards the Flows from the Traffic Control hooks in the eBPF kernel space.
// It provides access both to flows that are aggregated in the kernel space (via PerfCPU hashmap)
// and to flows that are forwarded by the kernel via ringbuffer because could not be aggregated
// in the map
type FlowFetcher struct {
	objects                  *BpfObjects
	qdiscs                   map[ifaces.Interface]*netlink.GenericQdisc
	egressFilters            map[ifaces.Interface]*netlink.BpfFilter
	ingressFilters           map[ifaces.Interface]*netlink.BpfFilter
	ringbufReader            *ringbuf.Reader
	cacheMaxSize             int
	enableIngress            bool
	enableEgress             bool
	pktDropsTracePoint       link.Link
	rttFentryLink            link.Link
	rttKprobeLink            link.Link
	egressTCXLink            map[ifaces.Interface]link.Link
	ingressTCXLink           map[ifaces.Interface]link.Link
	lookupAndDeleteSupported bool
}

type FlowFetcherConfig struct {
	EnableIngress    bool
	EnableEgress     bool
	Debug            bool
	Sampling         int
	CacheMaxSize     int
	PktDrops         bool
	DNSTracker       bool
	EnableRTT        bool
	EnableFlowFilter bool
	EnablePCA        bool
	FilterConfig     *FilterConfig
}

func NewFlowFetcher(cfg *FlowFetcherConfig) (*FlowFetcher, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.WithError(err).
			Warn("can't remove mem lock. The agent could not be able to start eBPF programs")
	}

	spec, err := LoadBpf()
	if err != nil {
		return nil, fmt.Errorf("loading BPF data: %w", err)
	}

	// Resize maps according to user-provided configuration
	spec.Maps[aggregatedFlowsMap].MaxEntries = uint32(cfg.CacheMaxSize)

	traceMsgs := 0
	if cfg.Debug {
		traceMsgs = 1
	}

	enableRtt := 0
	if cfg.EnableRTT {
		enableRtt = 1
	}

	enableDNSTracking := 0
	if cfg.DNSTracker {
		enableDNSTracking = 1
	}

	if enableDNSTracking == 0 {
		spec.Maps[dnsLatencyMap].MaxEntries = 1
	}

	enableFlowFiltering := 0
	if cfg.EnableFlowFilter {
		enableFlowFiltering = 1
	}

	if err := spec.RewriteConstants(map[string]interface{}{
		constSampling:            uint32(cfg.Sampling),
		constTraceMessages:       uint8(traceMsgs),
		constEnableRtt:           uint8(enableRtt),
		constEnableDNSTracking:   uint8(enableDNSTracking),
		constEnableFlowFiltering: uint8(enableFlowFiltering),
	}); err != nil {
		return nil, fmt.Errorf("rewriting BPF constants definition: %w", err)
	}

	oldKernel := kernel.IsKernelOlderThan("5.14.0")
	if oldKernel {
		log.Infof("kernel older than 5.14.0 detected: not all hooks are supported")
	}
	objects, err := kernelSpecificLoadAndAssign(oldKernel, spec)
	if err != nil {
		return nil, err
	}

	if cfg.EnableFlowFilter {
		f := NewFilter(&objects, cfg.FilterConfig)
		if err := f.ProgramFilter(); err != nil {
			return nil, fmt.Errorf("programming flow filter: %w", err)
		}
	}

	log.Debugf("Deleting specs for PCA")
	// Deleting specs for PCA
	// Always set pcaRecordsMap to the minimum in FlowFetcher - PCA and Flow Fetcher are mutually exclusive.
	spec.Maps[pcaRecordsMap].MaxEntries = 1

	objects.TcxEgressPcaParse = nil
	objects.TcIngressPcaParse = nil
	delete(spec.Programs, constPcaEnable)

	var pktDropsLink link.Link
	if cfg.PktDrops && !oldKernel {
		pktDropsLink, err = link.Tracepoint("skb", pktDropHook, objects.KfreeSkb, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to attach the BPF program to kfree_skb tracepoint: %w", err)
		}
	}

	var rttFentryLink, rttKprobeLink link.Link
	if cfg.EnableRTT {
		rttFentryLink, err = link.AttachTracing(link.TracingOptions{
			Program: objects.BpfPrograms.TcpRcvFentry,
		})
		if err != nil {
			log.Warningf("failed to attach the BPF program to tcpReceiveFentry: %v fallback to use kprobe", err)
			// try to use kprobe for older kernels
			rttKprobeLink, err = link.Kprobe("tcp_rcv_established", objects.TcpRcvKprobe, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to attach the BPF program to tcpReceiveKprobe: %w", err)
			}
		}
	}

	// read events from igress+egress ringbuffer
	flows, err := ringbuf.NewReader(objects.DirectFlows)
	if err != nil {
		return nil, fmt.Errorf("accessing to ringbuffer: %w", err)
	}

	return &FlowFetcher{
		objects:                  &objects,
		ringbufReader:            flows,
		egressFilters:            map[ifaces.Interface]*netlink.BpfFilter{},
		ingressFilters:           map[ifaces.Interface]*netlink.BpfFilter{},
		qdiscs:                   map[ifaces.Interface]*netlink.GenericQdisc{},
		cacheMaxSize:             cfg.CacheMaxSize,
		enableIngress:            cfg.EnableIngress,
		enableEgress:             cfg.EnableEgress,
		pktDropsTracePoint:       pktDropsLink,
		rttFentryLink:            rttFentryLink,
		rttKprobeLink:            rttKprobeLink,
		egressTCXLink:            map[ifaces.Interface]link.Link{},
		ingressTCXLink:           map[ifaces.Interface]link.Link{},
		lookupAndDeleteSupported: true, // this will be turned off later if found to be not supported
	}, nil
}

func (m *FlowFetcher) AttachTCX(iface ifaces.Interface) error {
	ilog := log.WithField("iface", iface)
	if iface.NetNS != netns.None() {
		originalNs, err := netns.Get()
		if err != nil {
			return fmt.Errorf("failed to get current netns: %w", err)
		}
		defer func() {
			if err := netns.Set(originalNs); err != nil {
				ilog.WithError(err).Error("failed to set netns back")
			}
			originalNs.Close()
		}()
		if err := unix.Setns(int(iface.NetNS), unix.CLONE_NEWNET); err != nil {
			return fmt.Errorf("failed to setns to %s: %w", iface.NetNS, err)
		}
	}

	if m.enableEgress {
		egrLink, err := link.AttachTCX(link.TCXOptions{
			Program:   m.objects.BpfPrograms.TcxEgressFlowParse,
			Attach:    ebpf.AttachTCXEgress,
			Interface: iface.Index,
		})
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				// The interface already has a TCX egress hook
				log.WithField("iface", iface.Name).Debug("interface already has a TCX egress hook ignore")
			} else {
				return fmt.Errorf("failed to attach TCX egress: %w", err)
			}
		}
		m.egressTCXLink[iface] = egrLink
		ilog.WithField("interface", iface.Name).Debug("successfully attach egressTCX hook")
	}

	if m.enableIngress {
		ingLink, err := link.AttachTCX(link.TCXOptions{
			Program:   m.objects.BpfPrograms.TcxIngressFlowParse,
			Attach:    ebpf.AttachTCXIngress,
			Interface: iface.Index,
		})
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				// The interface already has a TCX ingress hook
				log.WithField("iface", iface.Name).Debug("interface already has a TCX ingress hook ignore")
			} else {
				return fmt.Errorf("failed to attach TCX ingress: %w", err)
			}
		}
		m.ingressTCXLink[iface] = ingLink
		ilog.WithField("interface", iface.Name).Debug("successfully attach ingressTCX hook")
	}

	return nil
}

func removeTCFilters(ifName string, tcDir uint32) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return err
	}

	filters, err := netlink.FilterList(link, tcDir)
	if err != nil {
		return err
	}
	var errs []error
	for _, f := range filters {
		if err := netlink.FilterDel(f); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}

func (m *FlowFetcher) removePreviousFilters(iface ifaces.Interface) error {
	ilog := log.WithField("iface", iface)
	ilog.Debugf("looking for previously installed TC filters on %s", iface.Name)
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("retrieving all netlink devices: %w", err)
	}

	egressDevs := []netlink.Link{}
	ingressDevs := []netlink.Link{}
	for _, l := range links {
		if l.Attrs().Name != iface.Name {
			continue
		}
		ingressFilters, err := netlink.FilterList(l, netlink.HANDLE_MIN_INGRESS)
		if err != nil {
			return fmt.Errorf("listing ingress filters: %w", err)
		}
		for _, filter := range ingressFilters {
			if bpfFilter, ok := filter.(*netlink.BpfFilter); ok {
				if strings.HasPrefix(bpfFilter.Name, tcIngressFilterName) {
					ingressDevs = append(ingressDevs, l)
				}
			}
		}

		egressFilters, err := netlink.FilterList(l, netlink.HANDLE_MIN_EGRESS)
		if err != nil {
			return fmt.Errorf("listing egress filters: %w", err)
		}
		for _, filter := range egressFilters {
			if bpfFilter, ok := filter.(*netlink.BpfFilter); ok {
				if strings.HasPrefix(bpfFilter.Name, tcEgressFilterName) {
					egressDevs = append(egressDevs, l)
				}
			}
		}
	}

	for _, dev := range ingressDevs {
		ilog.Debugf("removing ingress stale tc filters from %s", dev.Attrs().Name)
		err = removeTCFilters(dev.Attrs().Name, netlink.HANDLE_MIN_INGRESS)
		if err != nil {
			ilog.WithError(err).Errorf("couldn't remove ingress tc filters from %s", dev.Attrs().Name)
		}
	}

	for _, dev := range egressDevs {
		ilog.Debugf("removing egress stale tc filters from %s", dev.Attrs().Name)
		err = removeTCFilters(dev.Attrs().Name, netlink.HANDLE_MIN_EGRESS)
		if err != nil {
			ilog.WithError(err).Errorf("couldn't remove egress tc filters from %s", dev.Attrs().Name)
		}
	}

	return nil
}

// Register and links the eBPF fetcher into the system. The program should invoke Unregister
// before exiting.
func (m *FlowFetcher) Register(iface ifaces.Interface) error {
	ilog := log.WithField("iface", iface)
	handle, err := netlink.NewHandleAt(iface.NetNS)
	if err != nil {
		return fmt.Errorf("failed to create handle for netns (%s): %w", iface.NetNS.String(), err)
	}
	defer handle.Delete()

	// Load pre-compiled programs and maps into the kernel, and rewrites the configuration
	ipvlan, err := handle.LinkByIndex(iface.Index)
	if err != nil {
		return fmt.Errorf("failed to lookup ipvlan device %d (%s): %w", iface.Index, iface.Name, err)
	}
	qdiscAttrs := netlink.QdiscAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Handle:    netlink.MakeHandle(0xffff, 0),
		Parent:    netlink.HANDLE_CLSACT,
	}
	qdisc := &netlink.GenericQdisc{
		QdiscAttrs: qdiscAttrs,
		QdiscType:  qdiscType,
	}
	if err := handle.QdiscDel(qdisc); err == nil {
		ilog.Warn("qdisc clsact already existed. Deleted it")
	}
	if err := handle.QdiscAdd(qdisc); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("qdisc clsact already exists. Ignoring")
		} else {
			return fmt.Errorf("failed to create clsact qdisc on %d (%s): %w", iface.Index, iface.Name, err)
		}
	}
	m.qdiscs[iface] = qdisc

	// Remove previously installed filters
	if err := m.removePreviousFilters(iface); err != nil {
		return fmt.Errorf("failed to remove previous filters: %w", err)
	}

	if err := m.registerEgress(iface, ipvlan, handle); err != nil {
		return err
	}

	return m.registerIngress(iface, ipvlan, handle)
}

func (m *FlowFetcher) registerEgress(iface ifaces.Interface, ipvlan netlink.Link, handle *netlink.Handle) error {
	ilog := log.WithField("iface", iface)
	if !m.enableEgress {
		ilog.Debug("ignoring egress traffic, according to user configuration")
		return nil
	}
	// Fetch events on egress
	egressAttrs := netlink.FilterAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Parent:    netlink.HANDLE_MIN_EGRESS,
		Handle:    netlink.MakeHandle(0, 1),
		Protocol:  3,
		Priority:  1,
	}
	egressFilter := &netlink.BpfFilter{
		FilterAttrs:  egressAttrs,
		Fd:           m.objects.TcEgressFlowParse.FD(),
		Name:         tcEgressFilterName,
		DirectAction: true,
	}
	if err := handle.FilterDel(egressFilter); err == nil {
		ilog.Warn("egress filter already existed. Deleted it")
	}
	if err := handle.FilterAdd(egressFilter); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("egress filter already exists. Ignoring")
		} else {
			return fmt.Errorf("failed to create egress filter: %w", err)
		}
	}
	m.egressFilters[iface] = egressFilter
	return nil
}

func (m *FlowFetcher) registerIngress(iface ifaces.Interface, ipvlan netlink.Link, handle *netlink.Handle) error {
	ilog := log.WithField("iface", iface)
	if !m.enableIngress {
		ilog.Debug("ignoring ingress traffic, according to user configuration")
		return nil
	}
	// Fetch events on ingress
	ingressAttrs := netlink.FilterAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Parent:    netlink.HANDLE_MIN_INGRESS,
		Handle:    netlink.MakeHandle(0, 1),
		Protocol:  unix.ETH_P_ALL,
		Priority:  1,
	}
	ingressFilter := &netlink.BpfFilter{
		FilterAttrs:  ingressAttrs,
		Fd:           m.objects.TcIngressFlowParse.FD(),
		Name:         tcIngressFilterName,
		DirectAction: true,
	}
	if err := handle.FilterDel(ingressFilter); err == nil {
		ilog.Warn("ingress filter already existed. Deleted it")
	}
	if err := handle.FilterAdd(ingressFilter); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("ingress filter already exists. Ignoring")
		} else {
			return fmt.Errorf("failed to create ingress filter: %w", err)
		}
	}
	m.ingressFilters[iface] = ingressFilter
	return nil
}

// Close the eBPF fetcher from the system.
// We don't need a "Close(iface)" method because the filters and qdiscs
// are automatically removed when the interface is down
// nolint:cyclop
func (m *FlowFetcher) Close() error {
	log.Debug("unregistering eBPF objects")

	var errs []error

	if m.pktDropsTracePoint != nil {
		if err := m.pktDropsTracePoint.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if m.rttFentryLink != nil {
		if err := m.rttFentryLink.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if m.rttKprobeLink != nil {
		if err := m.rttKprobeLink.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	// m.ringbufReader.Read is a blocking operation, so we need to close the ring buffer
	// from another goroutine to avoid the system not being able to exit if there
	// isn't traffic in a given interface
	if m.ringbufReader != nil {
		if err := m.ringbufReader.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if m.objects != nil {
		if err := m.objects.TcEgressFlowParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.TcIngressFlowParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.TcxEgressFlowParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.TcxIngressFlowParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.AggregatedFlows.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.DirectFlows.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.DnsFlows.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.GlobalCounters.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.FilterMap.Close(); err != nil {
			errs = append(errs, err)
		}
		if len(errs) == 0 {
			m.objects = nil
		}
	}

	for iface, ef := range m.egressFilters {
		log := log.WithField("interface", iface)
		log.Debug("deleting egress filter")
		if err := doIgnoreNoDev(netlink.FilterDel, netlink.Filter(ef), log); err != nil {
			errs = append(errs, fmt.Errorf("deleting egress filter: %w", err))
		}
	}
	m.egressFilters = map[ifaces.Interface]*netlink.BpfFilter{}
	for iface, igf := range m.ingressFilters {
		log := log.WithField("interface", iface)
		log.Debug("deleting ingress filter")
		if err := doIgnoreNoDev(netlink.FilterDel, netlink.Filter(igf), log); err != nil {
			errs = append(errs, fmt.Errorf("deleting ingress filter: %w", err))
		}
	}
	m.ingressFilters = map[ifaces.Interface]*netlink.BpfFilter{}
	for iface, qd := range m.qdiscs {
		log := log.WithField("interface", iface)
		log.Debug("deleting Qdisc")
		if err := doIgnoreNoDev(netlink.QdiscDel, netlink.Qdisc(qd), log); err != nil {
			errs = append(errs, fmt.Errorf("deleting qdisc: %w", err))
		}
	}
	m.qdiscs = map[ifaces.Interface]*netlink.GenericQdisc{}
	if len(errs) == 0 {
		return nil
	}
	for iface, l := range m.egressTCXLink {
		log := log.WithField("interface", iface)
		log.Debug("detach egress TCX hook")
		l.Close()
	}
	m.egressTCXLink = map[ifaces.Interface]link.Link{}
	for iface, l := range m.ingressTCXLink {
		log := log.WithField("interface", iface)
		log.Debug("detach ingress TCX hook")
		l.Close()
	}
	m.ingressTCXLink = map[ifaces.Interface]link.Link{}

	var errStrings []string
	for _, err := range errs {
		errStrings = append(errStrings, err.Error())
	}
	return errors.New(`errors: "` + strings.Join(errStrings, `", "`) + `"`)
}

// doIgnoreNoDev runs the provided syscall over the provided device and ignores the error
// if the cause is a non-existing device (just logs the error as debug).
// If the agent is deployed as part of the Network Observability pipeline, normally
// undeploying the FlowCollector could cause the agent to try to remove resources
// from Pods that have been removed immediately before (e.g. flowlogs-pipeline or the
// console plugin), so we avoid logging some errors that would unnecessarily raise the
// user's attention.
// This function uses generics because the set of provided functions accept different argument
// types.
func doIgnoreNoDev[T any](sysCall func(T) error, dev T, log *logrus.Entry) error {
	if err := sysCall(dev); err != nil {
		if errors.Is(err, unix.ENODEV) {
			log.WithError(err).Error("can't delete. Ignore this error if other pods or interfaces " +
				" are also being deleted at this moment. For example, if you are undeploying " +
				" a FlowCollector or Deployment where this agent is part of")
		} else {
			return err
		}
	}
	return nil
}

func (m *FlowFetcher) ReadRingBuf() (ringbuf.Record, error) {
	return m.ringbufReader.Read()
}

// LookupAndDeleteMap reads all the entries from the eBPF map and removes them from it.
// TODO: detect whether BatchLookupAndDelete is supported (Kernel>=5.6) and use it selectively
// Supported Lookup/Delete operations by kernel: https://github.com/iovisor/bcc/blob/master/docs/kernel-versions.md
func (m *FlowFetcher) LookupAndDeleteMap(met *metrics.Metrics) map[BpfFlowId][]BpfFlowMetrics {
	if !m.lookupAndDeleteSupported {
		return m.legacyLookupAndDeleteMap(met)
	}

	flowMap := m.objects.AggregatedFlows

	iterator := flowMap.Iterate()
	var flows = make(map[BpfFlowId][]BpfFlowMetrics, m.cacheMaxSize)
	var ids []BpfFlowId
	var id BpfFlowId
	var metrics []BpfFlowMetrics

	// First, get all ids and don't care about metrics (we need lookup+delete to be atomic)
	for iterator.Next(&id, &metrics) {
		ids = append(ids, id)
	}

	count := 0
	// Run the atomic Lookup+Delete; if new ids have been inserted in the meantime, they'll be fetched next time
	for i, id := range ids {
		count++
		if err := flowMap.LookupAndDelete(&id, &metrics); err != nil {
			if i == 0 && errors.Is(err, ebpf.ErrNotSupported) {
				log.WithError(err).Warnf("switching to legacy mode")
				m.lookupAndDeleteSupported = false
				return m.legacyLookupAndDeleteMap(met)
			}
			log.WithError(err).WithField("flowId", id).Warnf("couldn't delete flow entry")
			met.Errors.WithErrorName("flow-fetcher", "CannotDeleteFlows").Inc()
			continue
		}
		flows[id] = metrics
	}
	met.BufferSizeGauge.WithBufferName("hashmap-total").Set(float64(count))
	met.BufferSizeGauge.WithBufferName("hashmap-unique").Set(float64(len(flows)))

	m.ReadGlobalCounter(met)
	return flows
}

// ReadGlobalCounter reads the global counter and updates drop flows counter metrics
func (m *FlowFetcher) ReadGlobalCounter(met *metrics.Metrics) {
	var allCPUValue []uint32
	reasons := []string{
		"CannotUpdateHashMapCounter",
		"FilterRejectCounter",
		"FilterAcceptCounter",
		"FilterNoMatchCounter",
	}
	zeroCounters := make([]uint32, ebpf.MustPossibleCPU())
	for key := BpfGlobalCountersKeyTHASHMAP_FLOWS_DROPPED_KEY; key < BpfGlobalCountersKeyTMAX_DROPPED_FLOWS_KEY; key++ {
		if err := m.objects.GlobalCounters.Lookup(key, &allCPUValue); err != nil {
			log.WithError(err).Warnf("couldn't read global counter")
			return
		}
		// aggregate all the counters
		for _, counter := range allCPUValue {
			if key == BpfGlobalCountersKeyTHASHMAP_FLOWS_DROPPED_KEY {
				met.DroppedFlowsCounter.WithSourceAndReason("flow-fetcher", reasons[key]).Add(float64(counter))
			} else {
				met.FilteredFlowsCounter.WithSourceAndReason("flow-fetcher", reasons[key]).Add(float64(counter))
			}
		}
		// reset the global counter-map entry
		if err := m.objects.GlobalCounters.Put(key, zeroCounters); err != nil {
			log.WithError(err).Warnf("coudn't reset global counter")
			return
		}
	}
}

// DeleteMapsStaleEntries Look for any stale entries in the features maps and delete them
func (m *FlowFetcher) DeleteMapsStaleEntries(timeOut time.Duration) {
	m.lookupAndDeleteDNSMap(timeOut)
}

// lookupAndDeleteDNSMap iterate over DNS queries map and delete any stale DNS requests
// entries which never get responses for.
func (m *FlowFetcher) lookupAndDeleteDNSMap(timeOut time.Duration) {
	monotonicTimeNow := monotime.Now()
	dnsMap := m.objects.DnsFlows
	var dnsKey BpfDnsFlowId
	var keysToDelete []BpfDnsFlowId
	var dnsVal uint64

	if dnsMap != nil {
		// Ideally the Lookup + Delete should be atomic, however we cannot use LookupAndDelete since the deletion is conditional
		// Do not delete while iterating, as it causes severe performance degradation
		iterator := dnsMap.Iterate()
		for iterator.Next(&dnsKey, &dnsVal) {
			if time.Duration(uint64(monotonicTimeNow)-dnsVal) >= timeOut {
				keysToDelete = append(keysToDelete, dnsKey)
			}
		}
		for _, dnsKey = range keysToDelete {
			if err := dnsMap.Delete(dnsKey); err != nil {
				log.WithError(err).WithField("dnsKey", dnsKey).Warnf("couldn't delete DNS record entry")
			}
		}
	}
}

// kernelSpecificLoadAndAssign based on kernel version it will load only the supported ebPF hooks
func kernelSpecificLoadAndAssign(oldKernel bool, spec *ebpf.CollectionSpec) (BpfObjects, error) {
	objects := BpfObjects{}

	// For older kernel (< 5.14) kfree_sbk drop hook doesn't exists
	if oldKernel {
		// Here we define another structure similar to the bpf2go created one but w/o the hooks that does not exist in older kernel
		// Note: if new hooks are added in the future we need to update the following structures manually
		type NewBpfPrograms struct {
			TcEgressFlowParse   *ebpf.Program `ebpf:"tc_egress_flow_parse"`
			TcIngressFlowParse  *ebpf.Program `ebpf:"tc_ingress_flow_parse"`
			TcxEgressFlowParse  *ebpf.Program `ebpf:"tcx_egress_flow_parse"`
			TcxIngressFlowParse *ebpf.Program `ebpf:"tcx_ingress_flow_parse"`
			TcEgressPcaParse    *ebpf.Program `ebpf:"tc_egress_pca_parse"`
			TcIngressPcaParse   *ebpf.Program `ebpf:"tc_ingress_pca_parse"`
			TcxEgressPcaParse   *ebpf.Program `ebpf:"tcx_egress_pca_parse"`
			TcxIngressPcaParse  *ebpf.Program `ebpf:"tcx_ingress_pca_parse"`
			TCPRcvFentry        *ebpf.Program `ebpf:"tcp_rcv_fentry"`
			TCPRcvKprobe        *ebpf.Program `ebpf:"tcp_rcv_kprobe"`
		}
		type NewBpfObjects struct {
			NewBpfPrograms
			BpfMaps
		}
		var newObjects NewBpfObjects
		// remove pktdrop hook from the spec
		delete(spec.Programs, pktDropHook)
		newObjects.NewBpfPrograms = NewBpfPrograms{}
		if err := spec.LoadAndAssign(&newObjects, nil); err != nil {
			var ve *ebpf.VerifierError
			if errors.As(err, &ve) {
				// Using %+v will print the whole verifier error, not just the last
				// few lines.
				log.Infof("Verifier error: %+v", ve)
			}
			return objects, fmt.Errorf("loading and assigning BPF objects: %w", err)
		}
		// Manually assign maps and programs to the original objects variable
		// Note for any future maps or programs make sure to copy them manually here
		objects.DirectFlows = newObjects.DirectFlows
		objects.AggregatedFlows = newObjects.AggregatedFlows
		objects.DnsFlows = newObjects.DnsFlows
		objects.FilterMap = newObjects.FilterMap
		objects.GlobalCounters = newObjects.GlobalCounters
		objects.TcEgressFlowParse = newObjects.TcEgressFlowParse
		objects.TcIngressFlowParse = newObjects.TcIngressFlowParse
		objects.TcxEgressFlowParse = newObjects.TcxEgressFlowParse
		objects.TcxIngressFlowParse = newObjects.TcxIngressFlowParse
		objects.TcEgressPcaParse = newObjects.TcEgressPcaParse
		objects.TcIngressPcaParse = newObjects.TcIngressPcaParse
		objects.TcxEgressPcaParse = newObjects.TcxEgressPcaParse
		objects.TcxIngressPcaParse = newObjects.TcxIngressPcaParse
		objects.TcpRcvFentry = newObjects.TCPRcvFentry
		objects.TcpRcvKprobe = newObjects.TCPRcvKprobe
		objects.KfreeSkb = nil
	} else {
		if err := spec.LoadAndAssign(&objects, nil); err != nil {
			var ve *ebpf.VerifierError
			if errors.As(err, &ve) {
				// Using %+v will print the whole verifier error, not just the last
				// few lines.
				log.Infof("Verifier error: %+v", ve)
			}
			return objects, fmt.Errorf("loading and assigning BPF objects: %w", err)
		}
	}
	/*
	 * since we load the program only when the we start we need to release
	 * memory used by cached kernel BTF see https://github.com/cilium/ebpf/issues/1063
	 * for more details.
	 */
	btf.FlushKernelSpec()

	return objects, nil
}

// It provides access to packets from  the kernel space (via PerfCPU hashmap)
type PacketFetcher struct {
	objects                  *BpfObjects
	qdiscs                   map[ifaces.Interface]*netlink.GenericQdisc
	egressFilters            map[ifaces.Interface]*netlink.BpfFilter
	ingressFilters           map[ifaces.Interface]*netlink.BpfFilter
	perfReader               *perf.Reader
	cacheMaxSize             int
	enableIngress            bool
	enableEgress             bool
	egressTCXLink            map[ifaces.Interface]link.Link
	ingressTCXLink           map[ifaces.Interface]link.Link
	lookupAndDeleteSupported bool
}

func NewPacketFetcher(cfg *FlowFetcherConfig) (*PacketFetcher, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.WithError(err).
			Warn("can't remove mem lock. The agent could not be able to start eBPF programs")
	}

	objects := BpfObjects{}
	spec, err := LoadBpf()
	if err != nil {
		return nil, err
	}

	// Removing Specs for flow agent
	objects.TcEgressFlowParse = nil
	objects.TcIngressFlowParse = nil
	objects.TcxEgressFlowParse = nil
	objects.TcxIngressFlowParse = nil
	objects.DirectFlows = nil
	objects.AggregatedFlows = nil
	delete(spec.Programs, aggregatedFlowsMap)
	delete(spec.Programs, constSampling)
	delete(spec.Programs, constTraceMessages)
	delete(spec.Programs, constEnableDNSTracking)
	delete(spec.Programs, constEnableFlowFiltering)

	pcaEnable := 0
	if cfg.EnablePCA {
		pcaEnable = 1
	}

	if err := spec.RewriteConstants(map[string]interface{}{
		constSampling:  uint32(cfg.Sampling),
		constPcaEnable: uint8(pcaEnable),
	}); err != nil {
		return nil, fmt.Errorf("rewriting BPF constants definition: %w", err)
	}

	if err := spec.LoadAndAssign(&objects, nil); err != nil {
		var ve *ebpf.VerifierError
		if errors.As(err, &ve) {
			// Using %+v will print the whole verifier error, not just the last
			// few lines.
			plog.Infof("Verifier error: %+v", ve)
		}
		return nil, fmt.Errorf("loading and assigning BPF objects: %w", err)
	}

	f := NewFilter(&objects, cfg.FilterConfig)
	if err := f.ProgramFilter(); err != nil {
		return nil, fmt.Errorf("programming flow filter: %w", err)
	}

	// read packets from igress+egress perf array
	packets, err := perf.NewReader(objects.PacketRecord, os.Getpagesize())
	if err != nil {
		return nil, fmt.Errorf("accessing to perf: %w", err)
	}

	return &PacketFetcher{
		objects:                  &objects,
		perfReader:               packets,
		egressFilters:            map[ifaces.Interface]*netlink.BpfFilter{},
		ingressFilters:           map[ifaces.Interface]*netlink.BpfFilter{},
		qdiscs:                   map[ifaces.Interface]*netlink.GenericQdisc{},
		cacheMaxSize:             cfg.CacheMaxSize,
		enableIngress:            cfg.EnableIngress,
		enableEgress:             cfg.EnableEgress,
		egressTCXLink:            map[ifaces.Interface]link.Link{},
		ingressTCXLink:           map[ifaces.Interface]link.Link{},
		lookupAndDeleteSupported: true, // this will be turned off later if found to be not supported
	}, nil
}

func registerInterface(iface ifaces.Interface) (*netlink.GenericQdisc, netlink.Link, error) {
	ilog := plog.WithField("iface", iface)
	handle, err := netlink.NewHandleAt(iface.NetNS)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create handle for netns (%s): %w", iface.NetNS.String(), err)
	}
	defer handle.Delete()

	// Load pre-compiled programs and maps into the kernel, and rewrites the configuration
	ipvlan, err := handle.LinkByIndex(iface.Index)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup ipvlan device %d (%s): %w", iface.Index, iface.Name, err)
	}
	qdiscAttrs := netlink.QdiscAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Handle:    netlink.MakeHandle(0xffff, 0),
		Parent:    netlink.HANDLE_CLSACT,
	}
	qdisc := &netlink.GenericQdisc{
		QdiscAttrs: qdiscAttrs,
		QdiscType:  qdiscType,
	}
	if err := handle.QdiscDel(qdisc); err == nil {
		ilog.Warn("qdisc clsact already existed. Deleted it")
	}
	if err := handle.QdiscAdd(qdisc); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("qdisc clsact already exists. Ignoring")
		} else {
			return nil, nil, fmt.Errorf("failed to create clsact qdisc on %d (%s): %w", iface.Index, iface.Name, err)
		}
	}
	return qdisc, ipvlan, nil
}

func (p *PacketFetcher) Register(iface ifaces.Interface) error {

	qdisc, ipvlan, err := registerInterface(iface)
	if err != nil {
		return err
	}
	p.qdiscs[iface] = qdisc

	if err := p.registerEgress(iface, ipvlan); err != nil {
		return err
	}
	return p.registerIngress(iface, ipvlan)
}

func (p *PacketFetcher) AttachTCX(iface ifaces.Interface) error {
	ilog := log.WithField("iface", iface)
	if iface.NetNS != netns.None() {
		originalNs, err := netns.Get()
		if err != nil {
			return fmt.Errorf("PCA failed to get current netns: %w", err)
		}
		defer func() {
			if err := netns.Set(originalNs); err != nil {
				ilog.WithError(err).Error("PCA failed to set netns back")
			}
			originalNs.Close()
		}()
		if err := unix.Setns(int(iface.NetNS), unix.CLONE_NEWNET); err != nil {
			return fmt.Errorf("PCA failed to setns to %s: %w", iface.NetNS, err)
		}
	}

	if p.enableEgress {
		egrLink, err := link.AttachTCX(link.TCXOptions{
			Program:   p.objects.BpfPrograms.TcxEgressPcaParse,
			Attach:    ebpf.AttachTCXEgress,
			Interface: iface.Index,
		})
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				// The interface already has a TCX egress hook
				log.WithField("iface", iface.Name).Debug("interface already has a TCX PCA egress hook ignore")
			} else {
				return fmt.Errorf("failed to attach PCA TCX egress: %w", err)
			}
		}
		p.egressTCXLink[iface] = egrLink
		ilog.WithField("interface", iface.Name).Debug("successfully attach PCA egressTCX hook")
	}

	if p.enableIngress {
		ingLink, err := link.AttachTCX(link.TCXOptions{
			Program:   p.objects.BpfPrograms.TcxIngressPcaParse,
			Attach:    ebpf.AttachTCXIngress,
			Interface: iface.Index,
		})
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				// The interface already has a TCX ingress hook
				log.WithField("iface", iface.Name).Debug("interface already has a TCX PCA ingress hook ignore")
			} else {
				return fmt.Errorf("failed to attach PCA TCX ingress: %w", err)
			}
		}
		p.ingressTCXLink[iface] = ingLink
		ilog.WithField("interface", iface.Name).Debug("successfully attach PCA ingressTCX hook")
	}

	return nil
}

func fetchEgressEvents(iface ifaces.Interface, ipvlan netlink.Link, parser *ebpf.Program, name string) (*netlink.BpfFilter, error) {
	ilog := plog.WithField("iface", iface)
	egressAttrs := netlink.FilterAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Parent:    netlink.HANDLE_MIN_EGRESS,
		Handle:    netlink.MakeHandle(0, 1),
		Protocol:  3,
		Priority:  1,
	}
	egressFilter := &netlink.BpfFilter{
		FilterAttrs:  egressAttrs,
		Fd:           parser.FD(),
		Name:         "tc/" + name,
		DirectAction: true,
	}
	if err := netlink.FilterDel(egressFilter); err == nil {
		ilog.Warn("egress filter already existed. Deleted it")
	}
	if err := netlink.FilterAdd(egressFilter); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("egress filter already exists. Ignoring")
		} else {
			return nil, fmt.Errorf("failed to create egress filter: %w", err)
		}
	}
	return egressFilter, nil

}

func (p *PacketFetcher) registerEgress(iface ifaces.Interface, ipvlan netlink.Link) error {
	egressFilter, err := fetchEgressEvents(iface, ipvlan, p.objects.TcEgressPcaParse, "tc_egress_pca_parse")
	if err != nil {
		return err
	}

	p.egressFilters[iface] = egressFilter
	return nil
}

func fetchIngressEvents(iface ifaces.Interface, ipvlan netlink.Link, parser *ebpf.Program, name string) (*netlink.BpfFilter, error) {
	ilog := plog.WithField("iface", iface)
	ingressAttrs := netlink.FilterAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Parent:    netlink.HANDLE_MIN_INGRESS,
		Handle:    netlink.MakeHandle(0, 1),
		Protocol:  3,
		Priority:  1,
	}
	ingressFilter := &netlink.BpfFilter{
		FilterAttrs:  ingressAttrs,
		Fd:           parser.FD(),
		Name:         "tc/" + name,
		DirectAction: true,
	}
	if err := netlink.FilterDel(ingressFilter); err == nil {
		ilog.Warn("egress filter already existed. Deleted it")
	}
	if err := netlink.FilterAdd(ingressFilter); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("ingress filter already exists. Ignoring")
		} else {
			return nil, fmt.Errorf("failed to create egress filter: %w", err)
		}
	}
	return ingressFilter, nil

}

func (p *PacketFetcher) registerIngress(iface ifaces.Interface, ipvlan netlink.Link) error {
	ingressFilter, err := fetchIngressEvents(iface, ipvlan, p.objects.TcIngressPcaParse, "tc_ingress_pca_parse")
	if err != nil {
		return err
	}

	p.ingressFilters[iface] = ingressFilter
	return nil
}

// Close the eBPF fetcher from the system.
// We don't need an "Close(iface)" method because the filters and qdiscs
// are automatically removed when the interface is down
func (p *PacketFetcher) Close() error {
	log.Debug("unregistering eBPF objects")

	var errs []error
	if p.perfReader != nil {
		if err := p.perfReader.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.objects != nil {
		if err := p.objects.TcEgressPcaParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := p.objects.TcIngressPcaParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := p.objects.TcxEgressPcaParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := p.objects.TcxIngressPcaParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := p.objects.PacketRecord.Close(); err != nil {
			errs = append(errs, err)
		}
		p.objects = nil
	}
	for iface, ef := range p.egressFilters {
		log.WithField("interface", iface).Debug("deleting egress filter")
		if err := netlink.FilterDel(ef); err != nil {
			errs = append(errs, fmt.Errorf("deleting egress filter: %w", err))
		}
	}
	p.egressFilters = map[ifaces.Interface]*netlink.BpfFilter{}
	for iface, igf := range p.ingressFilters {
		log.WithField("interface", iface).Debug("deleting ingress filter")
		if err := netlink.FilterDel(igf); err != nil {
			errs = append(errs, fmt.Errorf("deleting ingress filter: %w", err))
		}
	}
	p.ingressFilters = map[ifaces.Interface]*netlink.BpfFilter{}
	for iface, qd := range p.qdiscs {
		log.WithField("interface", iface).Debug("deleting Qdisc")
		if err := netlink.QdiscDel(qd); err != nil {
			errs = append(errs, fmt.Errorf("deleting qdisc: %w", err))
		}
	}
	p.qdiscs = map[ifaces.Interface]*netlink.GenericQdisc{}
	if len(errs) == 0 {
		return nil
	}

	for iface, l := range p.egressTCXLink {
		log := log.WithField("interface", iface)
		log.Debug("detach egress TCX hook")
		l.Close()

	}
	p.egressTCXLink = map[ifaces.Interface]link.Link{}
	for iface, l := range p.ingressTCXLink {
		log := log.WithField("interface", iface)
		log.Debug("detach ingress TCX hook")
		l.Close()
	}
	p.ingressTCXLink = map[ifaces.Interface]link.Link{}

	var errStrings []string
	for _, err := range errs {
		errStrings = append(errStrings, err.Error())
	}
	return errors.New(`errors: "` + strings.Join(errStrings, `", "`) + `"`)
}

func (p *PacketFetcher) ReadPerf() (perf.Record, error) {
	return p.perfReader.Read()
}

func (p *PacketFetcher) LookupAndDeleteMap(met *metrics.Metrics) map[int][]*byte {
	if !p.lookupAndDeleteSupported {
		return p.legacyLookupAndDeleteMap(met)
	}

	packetMap := p.objects.PacketRecord
	iterator := packetMap.Iterate()
	packets := make(map[int][]*byte, p.cacheMaxSize)
	var id int
	var ids []int
	var packet []*byte

	// First, get all ids and ignore content (we need lookup+delete to be atomic)
	for iterator.Next(&id, &packet) {
		ids = append(ids, id)
	}

	// Run the atomic Lookup+Delete; if new ids have been inserted in the meantime, they'll be fetched next time
	for i, id := range ids {
		if err := packetMap.LookupAndDelete(&id, &packet); err != nil {
			if i == 0 && errors.Is(err, ebpf.ErrNotSupported) {
				log.WithError(err).Warnf("switching to legacy mode")
				p.lookupAndDeleteSupported = false
				return p.legacyLookupAndDeleteMap(met)
			}
			log.WithError(err).WithField("packetID", id).Warnf("couldn't delete entry")
			met.Errors.WithErrorName("pkt-fetcher", "CannotDeleteEntry").Inc()
		}
		packets[id] = packet
	}

	return packets
}
