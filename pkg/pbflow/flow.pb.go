// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.19.4
// source: proto/flow.proto

package pbflow

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// as defined by field 61 in
// https://www.iana.org/assignments/ipfix/ipfix.xhtml
type Direction int32

const (
	Direction_INGRESS Direction = 0
	Direction_EGRESS  Direction = 1
)

// Enum value maps for Direction.
var (
	Direction_name = map[int32]string{
		0: "INGRESS",
		1: "EGRESS",
	}
	Direction_value = map[string]int32{
		"INGRESS": 0,
		"EGRESS":  1,
	}
)

func (x Direction) Enum() *Direction {
	p := new(Direction)
	*p = x
	return p
}

func (x Direction) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Direction) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_flow_proto_enumTypes[0].Descriptor()
}

func (Direction) Type() protoreflect.EnumType {
	return &file_proto_flow_proto_enumTypes[0]
}

func (x Direction) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Direction.Descriptor instead.
func (Direction) EnumDescriptor() ([]byte, []int) {
	return file_proto_flow_proto_rawDescGZIP(), []int{0}
}

// intentionally empty
type CollectorReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *CollectorReply) Reset() {
	*x = CollectorReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_flow_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CollectorReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CollectorReply) ProtoMessage() {}

func (x *CollectorReply) ProtoReflect() protoreflect.Message {
	mi := &file_proto_flow_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CollectorReply.ProtoReflect.Descriptor instead.
func (*CollectorReply) Descriptor() ([]byte, []int) {
	return file_proto_flow_proto_rawDescGZIP(), []int{0}
}

type Records struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Entries []*Record `protobuf:"bytes,1,rep,name=entries,proto3" json:"entries,omitempty"`
}

func (x *Records) Reset() {
	*x = Records{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_flow_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Records) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Records) ProtoMessage() {}

func (x *Records) ProtoReflect() protoreflect.Message {
	mi := &file_proto_flow_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Records.ProtoReflect.Descriptor instead.
func (*Records) Descriptor() ([]byte, []int) {
	return file_proto_flow_proto_rawDescGZIP(), []int{1}
}

func (x *Records) GetEntries() []*Record {
	if x != nil {
		return x.Entries
	}
	return nil
}

type Record struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// protocol as defined by ETH_P_* in linux/if_ether.h
	// https://github.com/torvalds/linux/blob/master/include/uapi/linux/if_ether.h
	EthProtocol   uint32                 `protobuf:"varint,1,opt,name=eth_protocol,json=ethProtocol,proto3" json:"eth_protocol,omitempty"`
	Direction     Direction              `protobuf:"varint,2,opt,name=direction,proto3,enum=pbflow.Direction" json:"direction,omitempty"`
	TimeFlowStart *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=time_flow_start,json=timeFlowStart,proto3" json:"time_flow_start,omitempty"`
	TimeFlowEnd   *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=time_flow_end,json=timeFlowEnd,proto3" json:"time_flow_end,omitempty"`
	// OSI-layer attributes
	DataLink  *DataLink  `protobuf:"bytes,5,opt,name=data_link,json=dataLink,proto3" json:"data_link,omitempty"`
	Network   *Network   `protobuf:"bytes,6,opt,name=network,proto3" json:"network,omitempty"`
	Transport *Transport `protobuf:"bytes,7,opt,name=transport,proto3" json:"transport,omitempty"`
	Bytes     uint64     `protobuf:"varint,8,opt,name=bytes,proto3" json:"bytes,omitempty"`
	Packets   uint64     `protobuf:"varint,9,opt,name=packets,proto3" json:"packets,omitempty"`
	Interface string     `protobuf:"bytes,10,opt,name=interface,proto3" json:"interface,omitempty"`
	// if true, the same flow has been recorded via another interface.
	// From all the duplicate flows, one will set this value to false and the rest will be true.
	Duplicate bool `protobuf:"varint,11,opt,name=duplicate,proto3" json:"duplicate,omitempty"`
	// Agent IP address to help identifying the source of the flow
	AgentIp                *IP                  `protobuf:"bytes,12,opt,name=agent_ip,json=agentIp,proto3" json:"agent_ip,omitempty"`
	Flags                  uint32               `protobuf:"varint,13,opt,name=flags,proto3" json:"flags,omitempty"`
	IcmpType               uint32               `protobuf:"varint,14,opt,name=icmp_type,json=icmpType,proto3" json:"icmp_type,omitempty"`
	IcmpCode               uint32               `protobuf:"varint,15,opt,name=icmp_code,json=icmpCode,proto3" json:"icmp_code,omitempty"`
	PktDropBytes           uint64               `protobuf:"varint,16,opt,name=pkt_drop_bytes,json=pktDropBytes,proto3" json:"pkt_drop_bytes,omitempty"`
	PktDropPackets         uint64               `protobuf:"varint,17,opt,name=pkt_drop_packets,json=pktDropPackets,proto3" json:"pkt_drop_packets,omitempty"`
	PktDropLatestFlags     uint32               `protobuf:"varint,18,opt,name=pkt_drop_latest_flags,json=pktDropLatestFlags,proto3" json:"pkt_drop_latest_flags,omitempty"`
	PktDropLatestState     uint32               `protobuf:"varint,19,opt,name=pkt_drop_latest_state,json=pktDropLatestState,proto3" json:"pkt_drop_latest_state,omitempty"`
	PktDropLatestDropCause uint32               `protobuf:"varint,20,opt,name=pkt_drop_latest_drop_cause,json=pktDropLatestDropCause,proto3" json:"pkt_drop_latest_drop_cause,omitempty"`
	DnsId                  uint32               `protobuf:"varint,21,opt,name=dns_id,json=dnsId,proto3" json:"dns_id,omitempty"`
	DnsFlags               uint32               `protobuf:"varint,22,opt,name=dns_flags,json=dnsFlags,proto3" json:"dns_flags,omitempty"`
	DnsLatency             *durationpb.Duration `protobuf:"bytes,23,opt,name=dns_latency,json=dnsLatency,proto3" json:"dns_latency,omitempty"`
	TimeFlowRtt            *durationpb.Duration `protobuf:"bytes,24,opt,name=time_flow_rtt,json=timeFlowRtt,proto3" json:"time_flow_rtt,omitempty"`
	DnsErrno               uint32               `protobuf:"varint,25,opt,name=dns_errno,json=dnsErrno,proto3" json:"dns_errno,omitempty"`
}

func (x *Record) Reset() {
	*x = Record{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_flow_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Record) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Record) ProtoMessage() {}

func (x *Record) ProtoReflect() protoreflect.Message {
	mi := &file_proto_flow_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Record.ProtoReflect.Descriptor instead.
func (*Record) Descriptor() ([]byte, []int) {
	return file_proto_flow_proto_rawDescGZIP(), []int{2}
}

func (x *Record) GetEthProtocol() uint32 {
	if x != nil {
		return x.EthProtocol
	}
	return 0
}

func (x *Record) GetDirection() Direction {
	if x != nil {
		return x.Direction
	}
	return Direction_INGRESS
}

func (x *Record) GetTimeFlowStart() *timestamppb.Timestamp {
	if x != nil {
		return x.TimeFlowStart
	}
	return nil
}

func (x *Record) GetTimeFlowEnd() *timestamppb.Timestamp {
	if x != nil {
		return x.TimeFlowEnd
	}
	return nil
}

func (x *Record) GetDataLink() *DataLink {
	if x != nil {
		return x.DataLink
	}
	return nil
}

func (x *Record) GetNetwork() *Network {
	if x != nil {
		return x.Network
	}
	return nil
}

func (x *Record) GetTransport() *Transport {
	if x != nil {
		return x.Transport
	}
	return nil
}

func (x *Record) GetBytes() uint64 {
	if x != nil {
		return x.Bytes
	}
	return 0
}

func (x *Record) GetPackets() uint64 {
	if x != nil {
		return x.Packets
	}
	return 0
}

func (x *Record) GetInterface() string {
	if x != nil {
		return x.Interface
	}
	return ""
}

func (x *Record) GetDuplicate() bool {
	if x != nil {
		return x.Duplicate
	}
	return false
}

func (x *Record) GetAgentIp() *IP {
	if x != nil {
		return x.AgentIp
	}
	return nil
}

func (x *Record) GetFlags() uint32 {
	if x != nil {
		return x.Flags
	}
	return 0
}

func (x *Record) GetIcmpType() uint32 {
	if x != nil {
		return x.IcmpType
	}
	return 0
}

func (x *Record) GetIcmpCode() uint32 {
	if x != nil {
		return x.IcmpCode
	}
	return 0
}

func (x *Record) GetPktDropBytes() uint64 {
	if x != nil {
		return x.PktDropBytes
	}
	return 0
}

func (x *Record) GetPktDropPackets() uint64 {
	if x != nil {
		return x.PktDropPackets
	}
	return 0
}

func (x *Record) GetPktDropLatestFlags() uint32 {
	if x != nil {
		return x.PktDropLatestFlags
	}
	return 0
}

func (x *Record) GetPktDropLatestState() uint32 {
	if x != nil {
		return x.PktDropLatestState
	}
	return 0
}

func (x *Record) GetPktDropLatestDropCause() uint32 {
	if x != nil {
		return x.PktDropLatestDropCause
	}
	return 0
}

func (x *Record) GetDnsId() uint32 {
	if x != nil {
		return x.DnsId
	}
	return 0
}

func (x *Record) GetDnsFlags() uint32 {
	if x != nil {
		return x.DnsFlags
	}
	return 0
}

func (x *Record) GetDnsLatency() *durationpb.Duration {
	if x != nil {
		return x.DnsLatency
	}
	return nil
}

func (x *Record) GetTimeFlowRtt() *durationpb.Duration {
	if x != nil {
		return x.TimeFlowRtt
	}
	return nil
}

func (x *Record) GetDnsErrno() uint32 {
	if x != nil {
		return x.DnsErrno
	}
	return 0
}

type DataLink struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SrcMac uint64 `protobuf:"varint,1,opt,name=src_mac,json=srcMac,proto3" json:"src_mac,omitempty"`
	DstMac uint64 `protobuf:"varint,2,opt,name=dst_mac,json=dstMac,proto3" json:"dst_mac,omitempty"`
}

func (x *DataLink) Reset() {
	*x = DataLink{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_flow_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataLink) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataLink) ProtoMessage() {}

func (x *DataLink) ProtoReflect() protoreflect.Message {
	mi := &file_proto_flow_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataLink.ProtoReflect.Descriptor instead.
func (*DataLink) Descriptor() ([]byte, []int) {
	return file_proto_flow_proto_rawDescGZIP(), []int{3}
}

func (x *DataLink) GetSrcMac() uint64 {
	if x != nil {
		return x.SrcMac
	}
	return 0
}

func (x *DataLink) GetDstMac() uint64 {
	if x != nil {
		return x.DstMac
	}
	return 0
}

type Network struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SrcAddr *IP    `protobuf:"bytes,1,opt,name=src_addr,json=srcAddr,proto3" json:"src_addr,omitempty"`
	DstAddr *IP    `protobuf:"bytes,2,opt,name=dst_addr,json=dstAddr,proto3" json:"dst_addr,omitempty"`
	Dscp    uint32 `protobuf:"varint,3,opt,name=dscp,proto3" json:"dscp,omitempty"`
}

func (x *Network) Reset() {
	*x = Network{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_flow_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Network) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Network) ProtoMessage() {}

func (x *Network) ProtoReflect() protoreflect.Message {
	mi := &file_proto_flow_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Network.ProtoReflect.Descriptor instead.
func (*Network) Descriptor() ([]byte, []int) {
	return file_proto_flow_proto_rawDescGZIP(), []int{4}
}

func (x *Network) GetSrcAddr() *IP {
	if x != nil {
		return x.SrcAddr
	}
	return nil
}

func (x *Network) GetDstAddr() *IP {
	if x != nil {
		return x.DstAddr
	}
	return nil
}

func (x *Network) GetDscp() uint32 {
	if x != nil {
		return x.Dscp
	}
	return 0
}

type IP struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to IpFamily:
	//
	//	*IP_Ipv4
	//	*IP_Ipv6
	IpFamily isIP_IpFamily `protobuf_oneof:"ip_family"`
}

func (x *IP) Reset() {
	*x = IP{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_flow_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IP) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IP) ProtoMessage() {}

func (x *IP) ProtoReflect() protoreflect.Message {
	mi := &file_proto_flow_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IP.ProtoReflect.Descriptor instead.
func (*IP) Descriptor() ([]byte, []int) {
	return file_proto_flow_proto_rawDescGZIP(), []int{5}
}

func (m *IP) GetIpFamily() isIP_IpFamily {
	if m != nil {
		return m.IpFamily
	}
	return nil
}

func (x *IP) GetIpv4() uint32 {
	if x, ok := x.GetIpFamily().(*IP_Ipv4); ok {
		return x.Ipv4
	}
	return 0
}

func (x *IP) GetIpv6() []byte {
	if x, ok := x.GetIpFamily().(*IP_Ipv6); ok {
		return x.Ipv6
	}
	return nil
}

type isIP_IpFamily interface {
	isIP_IpFamily()
}

type IP_Ipv4 struct {
	Ipv4 uint32 `protobuf:"fixed32,1,opt,name=ipv4,proto3,oneof"`
}

type IP_Ipv6 struct {
	Ipv6 []byte `protobuf:"bytes,2,opt,name=ipv6,proto3,oneof"`
}

func (*IP_Ipv4) isIP_IpFamily() {}

func (*IP_Ipv6) isIP_IpFamily() {}

type Transport struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SrcPort uint32 `protobuf:"varint,1,opt,name=src_port,json=srcPort,proto3" json:"src_port,omitempty"`
	DstPort uint32 `protobuf:"varint,2,opt,name=dst_port,json=dstPort,proto3" json:"dst_port,omitempty"`
	// protocol as defined by IPPROTO_* in linux/in.h
	// https://github.com/torvalds/linux/blob/master/include/uapi/linux/in.h
	Protocol uint32 `protobuf:"varint,3,opt,name=protocol,proto3" json:"protocol,omitempty"`
}

func (x *Transport) Reset() {
	*x = Transport{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_flow_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Transport) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Transport) ProtoMessage() {}

func (x *Transport) ProtoReflect() protoreflect.Message {
	mi := &file_proto_flow_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Transport.ProtoReflect.Descriptor instead.
func (*Transport) Descriptor() ([]byte, []int) {
	return file_proto_flow_proto_rawDescGZIP(), []int{6}
}

func (x *Transport) GetSrcPort() uint32 {
	if x != nil {
		return x.SrcPort
	}
	return 0
}

func (x *Transport) GetDstPort() uint32 {
	if x != nil {
		return x.DstPort
	}
	return 0
}

func (x *Transport) GetProtocol() uint32 {
	if x != nil {
		return x.Protocol
	}
	return 0
}

var File_proto_flow_proto protoreflect.FileDescriptor

var file_proto_flow_proto_rawDesc = []byte{
	0x0a, 0x10, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x06, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x10, 0x0a, 0x0e, 0x43,
	0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x33, 0x0a,
	0x07, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x73, 0x12, 0x28, 0x0a, 0x07, 0x65, 0x6e, 0x74, 0x72,
	0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x62, 0x66, 0x6c,
	0x6f, 0x77, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x52, 0x07, 0x65, 0x6e, 0x74, 0x72, 0x69,
	0x65, 0x73, 0x22, 0x8c, 0x08, 0x0a, 0x06, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x21, 0x0a,
	0x0c, 0x65, 0x74, 0x68, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x0b, 0x65, 0x74, 0x68, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c,
	0x12, 0x2f, 0x0a, 0x09, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x44, 0x69, 0x72,
	0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x09, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x42, 0x0a, 0x0f, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x66, 0x6c, 0x6f, 0x77, 0x5f, 0x73,
	0x74, 0x61, 0x72, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0d, 0x74, 0x69, 0x6d, 0x65, 0x46, 0x6c, 0x6f, 0x77,
	0x53, 0x74, 0x61, 0x72, 0x74, 0x12, 0x3e, 0x0a, 0x0d, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x66, 0x6c,
	0x6f, 0x77, 0x5f, 0x65, 0x6e, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0b, 0x74, 0x69, 0x6d, 0x65, 0x46, 0x6c,
	0x6f, 0x77, 0x45, 0x6e, 0x64, 0x12, 0x2d, 0x0a, 0x09, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x6c, 0x69,
	0x6e, 0x6b, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x62, 0x66, 0x6c, 0x6f,
	0x77, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x4c, 0x69, 0x6e, 0x6b, 0x52, 0x08, 0x64, 0x61, 0x74, 0x61,
	0x4c, 0x69, 0x6e, 0x6b, 0x12, 0x29, 0x0a, 0x07, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x4e,
	0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x52, 0x07, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x12,
	0x2f, 0x0a, 0x09, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x54, 0x72, 0x61, 0x6e,
	0x73, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x09, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x70, 0x6f, 0x72, 0x74,
	0x12, 0x14, 0x0a, 0x05, 0x62, 0x79, 0x74, 0x65, 0x73, 0x18, 0x08, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x05, 0x62, 0x79, 0x74, 0x65, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x74,
	0x73, 0x18, 0x09, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x73,
	0x12, 0x1c, 0x0a, 0x09, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x18, 0x0a, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x12, 0x1c,
	0x0a, 0x09, 0x64, 0x75, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x65, 0x18, 0x0b, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x09, 0x64, 0x75, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x65, 0x12, 0x25, 0x0a, 0x08,
	0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x70, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a,
	0x2e, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x49, 0x50, 0x52, 0x07, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x49, 0x70, 0x12, 0x14, 0x0a, 0x05, 0x66, 0x6c, 0x61, 0x67, 0x73, 0x18, 0x0d, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x05, 0x66, 0x6c, 0x61, 0x67, 0x73, 0x12, 0x1b, 0x0a, 0x09, 0x69, 0x63, 0x6d,
	0x70, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x69, 0x63,
	0x6d, 0x70, 0x54, 0x79, 0x70, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x69, 0x63, 0x6d, 0x70, 0x5f, 0x63,
	0x6f, 0x64, 0x65, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x69, 0x63, 0x6d, 0x70, 0x43,
	0x6f, 0x64, 0x65, 0x12, 0x24, 0x0a, 0x0e, 0x70, 0x6b, 0x74, 0x5f, 0x64, 0x72, 0x6f, 0x70, 0x5f,
	0x62, 0x79, 0x74, 0x65, 0x73, 0x18, 0x10, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0c, 0x70, 0x6b, 0x74,
	0x44, 0x72, 0x6f, 0x70, 0x42, 0x79, 0x74, 0x65, 0x73, 0x12, 0x28, 0x0a, 0x10, 0x70, 0x6b, 0x74,
	0x5f, 0x64, 0x72, 0x6f, 0x70, 0x5f, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x73, 0x18, 0x11, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x0e, 0x70, 0x6b, 0x74, 0x44, 0x72, 0x6f, 0x70, 0x50, 0x61, 0x63, 0x6b,
	0x65, 0x74, 0x73, 0x12, 0x31, 0x0a, 0x15, 0x70, 0x6b, 0x74, 0x5f, 0x64, 0x72, 0x6f, 0x70, 0x5f,
	0x6c, 0x61, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x66, 0x6c, 0x61, 0x67, 0x73, 0x18, 0x12, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x12, 0x70, 0x6b, 0x74, 0x44, 0x72, 0x6f, 0x70, 0x4c, 0x61, 0x74, 0x65, 0x73,
	0x74, 0x46, 0x6c, 0x61, 0x67, 0x73, 0x12, 0x31, 0x0a, 0x15, 0x70, 0x6b, 0x74, 0x5f, 0x64, 0x72,
	0x6f, 0x70, 0x5f, 0x6c, 0x61, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18,
	0x13, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x12, 0x70, 0x6b, 0x74, 0x44, 0x72, 0x6f, 0x70, 0x4c, 0x61,
	0x74, 0x65, 0x73, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x3a, 0x0a, 0x1a, 0x70, 0x6b, 0x74,
	0x5f, 0x64, 0x72, 0x6f, 0x70, 0x5f, 0x6c, 0x61, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x64, 0x72, 0x6f,
	0x70, 0x5f, 0x63, 0x61, 0x75, 0x73, 0x65, 0x18, 0x14, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x16, 0x70,
	0x6b, 0x74, 0x44, 0x72, 0x6f, 0x70, 0x4c, 0x61, 0x74, 0x65, 0x73, 0x74, 0x44, 0x72, 0x6f, 0x70,
	0x43, 0x61, 0x75, 0x73, 0x65, 0x12, 0x15, 0x0a, 0x06, 0x64, 0x6e, 0x73, 0x5f, 0x69, 0x64, 0x18,
	0x15, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x64, 0x6e, 0x73, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09,
	0x64, 0x6e, 0x73, 0x5f, 0x66, 0x6c, 0x61, 0x67, 0x73, 0x18, 0x16, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x08, 0x64, 0x6e, 0x73, 0x46, 0x6c, 0x61, 0x67, 0x73, 0x12, 0x3a, 0x0a, 0x0b, 0x64, 0x6e, 0x73,
	0x5f, 0x6c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79, 0x18, 0x17, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x64, 0x6e, 0x73, 0x4c, 0x61,
	0x74, 0x65, 0x6e, 0x63, 0x79, 0x12, 0x3d, 0x0a, 0x0d, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x66, 0x6c,
	0x6f, 0x77, 0x5f, 0x72, 0x74, 0x74, 0x18, 0x18, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0b, 0x74, 0x69, 0x6d, 0x65, 0x46, 0x6c, 0x6f,
	0x77, 0x52, 0x74, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x64, 0x6e, 0x73, 0x5f, 0x65, 0x72, 0x72, 0x6e,
	0x6f, 0x18, 0x19, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x64, 0x6e, 0x73, 0x45, 0x72, 0x72, 0x6e,
	0x6f, 0x22, 0x3c, 0x0a, 0x08, 0x44, 0x61, 0x74, 0x61, 0x4c, 0x69, 0x6e, 0x6b, 0x12, 0x17, 0x0a,
	0x07, 0x73, 0x72, 0x63, 0x5f, 0x6d, 0x61, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06,
	0x73, 0x72, 0x63, 0x4d, 0x61, 0x63, 0x12, 0x17, 0x0a, 0x07, 0x64, 0x73, 0x74, 0x5f, 0x6d, 0x61,
	0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x64, 0x73, 0x74, 0x4d, 0x61, 0x63, 0x22,
	0x6b, 0x0a, 0x07, 0x4e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x12, 0x25, 0x0a, 0x08, 0x73, 0x72,
	0x63, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x70,
	0x62, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x49, 0x50, 0x52, 0x07, 0x73, 0x72, 0x63, 0x41, 0x64, 0x64,
	0x72, 0x12, 0x25, 0x0a, 0x08, 0x64, 0x73, 0x74, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x49, 0x50, 0x52,
	0x07, 0x64, 0x73, 0x74, 0x41, 0x64, 0x64, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x73, 0x63, 0x70,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x64, 0x73, 0x63, 0x70, 0x22, 0x3d, 0x0a, 0x02,
	0x49, 0x50, 0x12, 0x14, 0x0a, 0x04, 0x69, 0x70, 0x76, 0x34, 0x18, 0x01, 0x20, 0x01, 0x28, 0x07,
	0x48, 0x00, 0x52, 0x04, 0x69, 0x70, 0x76, 0x34, 0x12, 0x14, 0x0a, 0x04, 0x69, 0x70, 0x76, 0x36,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x48, 0x00, 0x52, 0x04, 0x69, 0x70, 0x76, 0x36, 0x42, 0x0b,
	0x0a, 0x09, 0x69, 0x70, 0x5f, 0x66, 0x61, 0x6d, 0x69, 0x6c, 0x79, 0x22, 0x5d, 0x0a, 0x09, 0x54,
	0x72, 0x61, 0x6e, 0x73, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x73, 0x72, 0x63, 0x5f,
	0x70, 0x6f, 0x72, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x73, 0x72, 0x63, 0x50,
	0x6f, 0x72, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x64, 0x73, 0x74, 0x5f, 0x70, 0x6f, 0x72, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x64, 0x73, 0x74, 0x50, 0x6f, 0x72, 0x74, 0x12, 0x1a,
	0x0a, 0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2a, 0x24, 0x0a, 0x09, 0x44, 0x69,
	0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x0b, 0x0a, 0x07, 0x49, 0x4e, 0x47, 0x52, 0x45,
	0x53, 0x53, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06, 0x45, 0x47, 0x52, 0x45, 0x53, 0x53, 0x10, 0x01,
	0x32, 0x3e, 0x0a, 0x09, 0x43, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x31, 0x0a,
	0x04, 0x53, 0x65, 0x6e, 0x64, 0x12, 0x0f, 0x2e, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x52,
	0x65, 0x63, 0x6f, 0x72, 0x64, 0x73, 0x1a, 0x16, 0x2e, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x2e,
	0x43, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00,
	0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x2f, 0x70, 0x62, 0x66, 0x6c, 0x6f, 0x77, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_flow_proto_rawDescOnce sync.Once
	file_proto_flow_proto_rawDescData = file_proto_flow_proto_rawDesc
)

func file_proto_flow_proto_rawDescGZIP() []byte {
	file_proto_flow_proto_rawDescOnce.Do(func() {
		file_proto_flow_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_flow_proto_rawDescData)
	})
	return file_proto_flow_proto_rawDescData
}

var file_proto_flow_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_flow_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_proto_flow_proto_goTypes = []interface{}{
	(Direction)(0),                // 0: pbflow.Direction
	(*CollectorReply)(nil),        // 1: pbflow.CollectorReply
	(*Records)(nil),               // 2: pbflow.Records
	(*Record)(nil),                // 3: pbflow.Record
	(*DataLink)(nil),              // 4: pbflow.DataLink
	(*Network)(nil),               // 5: pbflow.Network
	(*IP)(nil),                    // 6: pbflow.IP
	(*Transport)(nil),             // 7: pbflow.Transport
	(*timestamppb.Timestamp)(nil), // 8: google.protobuf.Timestamp
	(*durationpb.Duration)(nil),   // 9: google.protobuf.Duration
}
var file_proto_flow_proto_depIdxs = []int32{
	3,  // 0: pbflow.Records.entries:type_name -> pbflow.Record
	0,  // 1: pbflow.Record.direction:type_name -> pbflow.Direction
	8,  // 2: pbflow.Record.time_flow_start:type_name -> google.protobuf.Timestamp
	8,  // 3: pbflow.Record.time_flow_end:type_name -> google.protobuf.Timestamp
	4,  // 4: pbflow.Record.data_link:type_name -> pbflow.DataLink
	5,  // 5: pbflow.Record.network:type_name -> pbflow.Network
	7,  // 6: pbflow.Record.transport:type_name -> pbflow.Transport
	6,  // 7: pbflow.Record.agent_ip:type_name -> pbflow.IP
	9,  // 8: pbflow.Record.dns_latency:type_name -> google.protobuf.Duration
	9,  // 9: pbflow.Record.time_flow_rtt:type_name -> google.protobuf.Duration
	6,  // 10: pbflow.Network.src_addr:type_name -> pbflow.IP
	6,  // 11: pbflow.Network.dst_addr:type_name -> pbflow.IP
	2,  // 12: pbflow.Collector.Send:input_type -> pbflow.Records
	1,  // 13: pbflow.Collector.Send:output_type -> pbflow.CollectorReply
	13, // [13:14] is the sub-list for method output_type
	12, // [12:13] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_proto_flow_proto_init() }
func file_proto_flow_proto_init() {
	if File_proto_flow_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_flow_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CollectorReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_flow_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Records); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_flow_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Record); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_flow_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataLink); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_flow_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Network); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_flow_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IP); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_flow_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Transport); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_proto_flow_proto_msgTypes[5].OneofWrappers = []interface{}{
		(*IP_Ipv4)(nil),
		(*IP_Ipv6)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_flow_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_flow_proto_goTypes,
		DependencyIndexes: file_proto_flow_proto_depIdxs,
		EnumInfos:         file_proto_flow_proto_enumTypes,
		MessageInfos:      file_proto_flow_proto_msgTypes,
	}.Build()
	File_proto_flow_proto = out.File
	file_proto_flow_proto_rawDesc = nil
	file_proto_flow_proto_goTypes = nil
	file_proto_flow_proto_depIdxs = nil
}
