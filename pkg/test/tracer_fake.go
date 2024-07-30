package test

import (
	"bytes"
	"encoding/binary"
	"github.com/cilium/ebpf/perf"
	"time"

	"github.com/netobserv/netobserv-ebpf-agent/pkg/ebpf"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/flow"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/ifaces"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/metrics"

	"github.com/cilium/ebpf/ringbuf"
)

// TracerFake fakes the kernel-side eBPF map structures for testing
type TracerFake struct {
	interfaces map[ifaces.Interface]struct{}
	mapLookups chan map[ebpf.BpfFlowId][]ebpf.BpfFlowMetrics
	ringBuf    chan ringbuf.Record
	perf       chan perf.Record
}

func NewTracerFake() *TracerFake {
	return &TracerFake{
		interfaces: map[ifaces.Interface]struct{}{},
		mapLookups: make(chan map[ebpf.BpfFlowId][]ebpf.BpfFlowMetrics, 100),
		ringBuf:    make(chan ringbuf.Record, 100),
		perf:       make(chan perf.Record, 100),
	}
}

func (m *TracerFake) Close() error {
	return nil
}
func (m *TracerFake) Register(iface ifaces.Interface) error {
	m.interfaces[iface] = struct{}{}
	return nil
}

func (m *TracerFake) AttachTCX(_ ifaces.Interface) error {
	return nil
}

func (m *TracerFake) LookupAndDeleteMap(_ *metrics.Metrics) map[ebpf.BpfFlowId][]ebpf.BpfFlowMetrics {
	select {
	case r := <-m.mapLookups:
		return r
	default:
		return map[ebpf.BpfFlowId][]ebpf.BpfFlowMetrics{}
	}
}

func (m *TracerFake) DeleteMapsStaleEntries(_ time.Duration) {
}

func (m *TracerFake) ReadPerf() (perf.Record, error) {
	return <-m.perf, nil
}

func (m *TracerFake) AppendLookupResults(results map[ebpf.BpfFlowId][]ebpf.BpfFlowMetrics) {
	m.mapLookups <- results
}

//nolint:gocritic // we don't care about efficiency of a large argument in test fakes
func (m *TracerFake) AppendRingBufEvent(flow flow.RawRecord) error {
	encodedRecord := bytes.Buffer{}
	if err := binary.Write(&encodedRecord, binary.LittleEndian, flow); err != nil {
		return err
	}
	m.ringBuf <- ringbuf.Record{RawSample: encodedRecord.Bytes()}
	return nil
}
