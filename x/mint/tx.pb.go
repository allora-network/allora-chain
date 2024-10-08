// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: mint/v2/tx.proto

package mint

import (
	context "context"
	fmt "fmt"
	types "github.com/allora-network/allora-chain/x/mint/types"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// MsgUpdateParams allows an update to the minting parameters of the module.
type MsgServiceUpdateParamsRequest struct {
	Sender string `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	// params defines the x/mint parameters to update.
	//
	// NOTE: All parameters must be supplied.
	Params                    types.Params `protobuf:"bytes,2,opt,name=params,proto3" json:"params"`
	RecalculateTargetEmission bool         `protobuf:"varint,3,opt,name=recalculate_target_emission,json=recalculateTargetEmission,proto3" json:"recalculate_target_emission,omitempty"`
}

func (m *MsgServiceUpdateParamsRequest) Reset()         { *m = MsgServiceUpdateParamsRequest{} }
func (m *MsgServiceUpdateParamsRequest) String() string { return proto.CompactTextString(m) }
func (*MsgServiceUpdateParamsRequest) ProtoMessage()    {}
func (*MsgServiceUpdateParamsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2bf02b1ff3ccb0c3, []int{0}
}
func (m *MsgServiceUpdateParamsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgServiceUpdateParamsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgServiceUpdateParamsRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgServiceUpdateParamsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgServiceUpdateParamsRequest.Merge(m, src)
}
func (m *MsgServiceUpdateParamsRequest) XXX_Size() int {
	return m.Size()
}
func (m *MsgServiceUpdateParamsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgServiceUpdateParamsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_MsgServiceUpdateParamsRequest proto.InternalMessageInfo

func (m *MsgServiceUpdateParamsRequest) GetSender() string {
	if m != nil {
		return m.Sender
	}
	return ""
}

func (m *MsgServiceUpdateParamsRequest) GetParams() types.Params {
	if m != nil {
		return m.Params
	}
	return types.Params{}
}

func (m *MsgServiceUpdateParamsRequest) GetRecalculateTargetEmission() bool {
	if m != nil {
		return m.RecalculateTargetEmission
	}
	return false
}

// MsgUpdateParamsResponse defines the response structure for executing a
// MsgUpdateParams message.
type MsgServiceUpdateParamsResponse struct {
}

func (m *MsgServiceUpdateParamsResponse) Reset()         { *m = MsgServiceUpdateParamsResponse{} }
func (m *MsgServiceUpdateParamsResponse) String() string { return proto.CompactTextString(m) }
func (*MsgServiceUpdateParamsResponse) ProtoMessage()    {}
func (*MsgServiceUpdateParamsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_2bf02b1ff3ccb0c3, []int{1}
}
func (m *MsgServiceUpdateParamsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgServiceUpdateParamsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgServiceUpdateParamsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgServiceUpdateParamsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgServiceUpdateParamsResponse.Merge(m, src)
}
func (m *MsgServiceUpdateParamsResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgServiceUpdateParamsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgServiceUpdateParamsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgServiceUpdateParamsResponse proto.InternalMessageInfo

// Force a recalculation of the target emission right now.
// This indirectly controls recalculating the inflation rate for the network
// and the stakers APY %.
type MsgServiceRecalculateTargetEmissionRequest struct {
	Sender string `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
}

func (m *MsgServiceRecalculateTargetEmissionRequest) Reset() {
	*m = MsgServiceRecalculateTargetEmissionRequest{}
}
func (m *MsgServiceRecalculateTargetEmissionRequest) String() string {
	return proto.CompactTextString(m)
}
func (*MsgServiceRecalculateTargetEmissionRequest) ProtoMessage() {}
func (*MsgServiceRecalculateTargetEmissionRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2bf02b1ff3ccb0c3, []int{2}
}
func (m *MsgServiceRecalculateTargetEmissionRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgServiceRecalculateTargetEmissionRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgServiceRecalculateTargetEmissionRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgServiceRecalculateTargetEmissionRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgServiceRecalculateTargetEmissionRequest.Merge(m, src)
}
func (m *MsgServiceRecalculateTargetEmissionRequest) XXX_Size() int {
	return m.Size()
}
func (m *MsgServiceRecalculateTargetEmissionRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgServiceRecalculateTargetEmissionRequest.DiscardUnknown(m)
}

var xxx_messageInfo_MsgServiceRecalculateTargetEmissionRequest proto.InternalMessageInfo

func (m *MsgServiceRecalculateTargetEmissionRequest) GetSender() string {
	if m != nil {
		return m.Sender
	}
	return ""
}

// response from recalculating the target emission
type MsgServiceRecalculateTargetEmissionResponse struct {
}

func (m *MsgServiceRecalculateTargetEmissionResponse) Reset() {
	*m = MsgServiceRecalculateTargetEmissionResponse{}
}
func (m *MsgServiceRecalculateTargetEmissionResponse) String() string {
	return proto.CompactTextString(m)
}
func (*MsgServiceRecalculateTargetEmissionResponse) ProtoMessage() {}
func (*MsgServiceRecalculateTargetEmissionResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_2bf02b1ff3ccb0c3, []int{3}
}
func (m *MsgServiceRecalculateTargetEmissionResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgServiceRecalculateTargetEmissionResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgServiceRecalculateTargetEmissionResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgServiceRecalculateTargetEmissionResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgServiceRecalculateTargetEmissionResponse.Merge(m, src)
}
func (m *MsgServiceRecalculateTargetEmissionResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgServiceRecalculateTargetEmissionResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgServiceRecalculateTargetEmissionResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgServiceRecalculateTargetEmissionResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgServiceUpdateParamsRequest)(nil), "mint.v2.MsgServiceUpdateParamsRequest")
	proto.RegisterType((*MsgServiceUpdateParamsResponse)(nil), "mint.v2.MsgServiceUpdateParamsResponse")
	proto.RegisterType((*MsgServiceRecalculateTargetEmissionRequest)(nil), "mint.v2.MsgServiceRecalculateTargetEmissionRequest")
	proto.RegisterType((*MsgServiceRecalculateTargetEmissionResponse)(nil), "mint.v2.MsgServiceRecalculateTargetEmissionResponse")
}

func init() { proto.RegisterFile("mint/v2/tx.proto", fileDescriptor_2bf02b1ff3ccb0c3) }

var fileDescriptor_2bf02b1ff3ccb0c3 = []byte{
	// 469 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x93, 0x31, 0x8b, 0x13, 0x41,
	0x14, 0xc7, 0x33, 0x8a, 0xd1, 0x1b, 0x2d, 0x74, 0x09, 0xb8, 0x59, 0x71, 0x0d, 0x11, 0x34, 0x44,
	0xb2, 0x73, 0xd9, 0x08, 0xc2, 0x15, 0xa2, 0x01, 0xb1, 0x3a, 0x90, 0x3d, 0x6d, 0x6c, 0xc2, 0x64,
	0xf3, 0x98, 0x1b, 0xcc, 0xce, 0xac, 0x33, 0x93, 0x78, 0x76, 0x62, 0x61, 0x61, 0xe5, 0x47, 0xb0,
	0xb4, 0x4c, 0xe1, 0x87, 0xb8, 0xf2, 0xb0, 0xb2, 0x12, 0x49, 0x8a, 0x7c, 0x0a, 0x51, 0x76, 0x67,
	0xe0, 0x82, 0xdc, 0x7a, 0x6a, 0x93, 0xec, 0xbc, 0xf7, 0xdb, 0xf7, 0xff, 0xbf, 0xf7, 0x66, 0xf1,
	0xe5, 0x8c, 0x0b, 0x43, 0xe6, 0x31, 0x31, 0x07, 0x51, 0xae, 0xa4, 0x91, 0xde, 0xf9, 0x22, 0x12,
	0xcd, 0xe3, 0xe0, 0x0a, 0xcd, 0xb8, 0x90, 0xa4, 0xfc, 0xb5, 0xb9, 0xe0, 0x6a, 0x2a, 0x75, 0x26,
	0x35, 0xc9, 0x34, 0x23, 0xf3, 0x7e, 0xf1, 0xe7, 0x12, 0x4d, 0x9b, 0x18, 0x95, 0x27, 0x62, 0x0f,
	0x2e, 0xd5, 0x60, 0x92, 0x49, 0x1b, 0x2f, 0x9e, 0x5c, 0xd4, 0xb7, 0xba, 0xfd, 0x31, 0x18, 0xda,
	0x27, 0xe6, 0x75, 0x0e, 0x8e, 0x6f, 0xff, 0x44, 0xf8, 0xfa, 0xae, 0x66, 0x7b, 0xa0, 0xe6, 0x3c,
	0x85, 0x67, 0xf9, 0x84, 0x1a, 0x78, 0x42, 0x15, 0xcd, 0x74, 0x02, 0x2f, 0x67, 0xa0, 0x8d, 0xb7,
	0x8d, 0xeb, 0x1a, 0xc4, 0x04, 0x94, 0x8f, 0x5a, 0xa8, 0xb3, 0x35, 0xf4, 0xbf, 0x7c, 0xee, 0x35,
	0x9c, 0xe6, 0xc3, 0xc9, 0x44, 0x81, 0xd6, 0x7b, 0x46, 0x71, 0xc1, 0x12, 0xc7, 0x79, 0xf7, 0x70,
	0x3d, 0x2f, 0x4b, 0xf8, 0x67, 0x5a, 0xa8, 0x73, 0x31, 0x6e, 0x44, 0xb6, 0x49, 0x2b, 0x1f, 0xd9,
	0xf2, 0xc3, 0xad, 0xc3, 0x6f, 0x37, 0x6a, 0x9f, 0xd6, 0x8b, 0x2e, 0x4a, 0x1c, 0xee, 0xdd, 0xc7,
	0xd7, 0x14, 0xa4, 0x74, 0x9a, 0xce, 0xa6, 0xd4, 0xc0, 0xc8, 0x50, 0xc5, 0xc0, 0x8c, 0x20, 0xe3,
	0x5a, 0x73, 0x29, 0xfc, 0xb3, 0x2d, 0xd4, 0xb9, 0x90, 0x34, 0x37, 0x90, 0xa7, 0x25, 0xf1, 0xc8,
	0x01, 0x3b, 0x83, 0xb7, 0xeb, 0x45, 0xd7, 0xb9, 0x78, 0xbf, 0x5e, 0x74, 0x6f, 0xd2, 0xe9, 0x54,
	0x2a, 0xda, 0x4b, 0xf7, 0x29, 0x17, 0xe4, 0x80, 0x94, 0x53, 0xd8, 0xd5, 0x6c, 0xb3, 0xcd, 0x76,
	0x0b, 0x87, 0x55, 0x03, 0xd0, 0xb9, 0x14, 0x1a, 0xda, 0x1f, 0x11, 0xee, 0x1e, 0x23, 0x49, 0x95,
	0xfc, 0x7f, 0x0f, 0x6c, 0xe7, 0xc1, 0x6f, 0xbe, 0xb7, 0x2b, 0x7c, 0x57, 0x4a, 0xb7, 0x7b, 0xf8,
	0xce, 0x5f, 0x39, 0xb4, 0x1d, 0xc5, 0x3f, 0x10, 0xc6, 0xc7, 0xbc, 0x37, 0xc2, 0x97, 0x36, 0x1b,
	0xf7, 0x6e, 0xb9, 0x85, 0xc5, 0xd1, 0x1f, 0xaf, 0x46, 0x70, 0xfb, 0x54, 0xce, 0xea, 0x79, 0xef,
	0x10, 0x6e, 0x56, 0xba, 0xf2, 0x06, 0x27, 0x94, 0x39, 0x6d, 0xca, 0xc1, 0xdd, 0x7f, 0x7b, 0xc9,
	0x1a, 0x09, 0xce, 0xbd, 0x29, 0x2e, 0xdc, 0xf0, 0xf1, 0xe1, 0x32, 0x44, 0x47, 0xcb, 0x10, 0x7d,
	0x5f, 0x86, 0xe8, 0xc3, 0x2a, 0xac, 0x1d, 0xad, 0xc2, 0xda, 0xd7, 0x55, 0x58, 0x7b, 0xde, 0x63,
	0xdc, 0xec, 0xcf, 0xc6, 0x51, 0x2a, 0x33, 0xe2, 0xb6, 0x20, 0xc0, 0xbc, 0x92, 0xea, 0x05, 0x39,
	0x61, 0x29, 0xe3, 0x7a, 0xf9, 0x15, 0x0d, 0x7e, 0x05, 0x00, 0x00, 0xff, 0xff, 0x85, 0x45, 0x8d,
	0xe5, 0xd9, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgServiceClient is the client API for MsgService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgServiceClient interface {
	// update params. Only callable by someone on the emissions module whitelist
	UpdateParams(ctx context.Context, in *MsgServiceUpdateParamsRequest, opts ...grpc.CallOption) (*MsgServiceUpdateParamsResponse, error)
	// force a target emission calculation right now. Otherwise waits until the
	// end of params.BlocksPerMonth
	RecalculateTargetEmission(ctx context.Context, in *MsgServiceRecalculateTargetEmissionRequest, opts ...grpc.CallOption) (*MsgServiceRecalculateTargetEmissionResponse, error)
}

type msgServiceClient struct {
	cc grpc1.ClientConn
}

func NewMsgServiceClient(cc grpc1.ClientConn) MsgServiceClient {
	return &msgServiceClient{cc}
}

func (c *msgServiceClient) UpdateParams(ctx context.Context, in *MsgServiceUpdateParamsRequest, opts ...grpc.CallOption) (*MsgServiceUpdateParamsResponse, error) {
	out := new(MsgServiceUpdateParamsResponse)
	err := c.cc.Invoke(ctx, "/mint.v2.MsgService/UpdateParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgServiceClient) RecalculateTargetEmission(ctx context.Context, in *MsgServiceRecalculateTargetEmissionRequest, opts ...grpc.CallOption) (*MsgServiceRecalculateTargetEmissionResponse, error) {
	out := new(MsgServiceRecalculateTargetEmissionResponse)
	err := c.cc.Invoke(ctx, "/mint.v2.MsgService/RecalculateTargetEmission", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServiceServer is the server API for MsgService service.
type MsgServiceServer interface {
	// update params. Only callable by someone on the emissions module whitelist
	UpdateParams(context.Context, *MsgServiceUpdateParamsRequest) (*MsgServiceUpdateParamsResponse, error)
	// force a target emission calculation right now. Otherwise waits until the
	// end of params.BlocksPerMonth
	RecalculateTargetEmission(context.Context, *MsgServiceRecalculateTargetEmissionRequest) (*MsgServiceRecalculateTargetEmissionResponse, error)
}

// UnimplementedMsgServiceServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServiceServer struct {
}

func (*UnimplementedMsgServiceServer) UpdateParams(ctx context.Context, req *MsgServiceUpdateParamsRequest) (*MsgServiceUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParams not implemented")
}
func (*UnimplementedMsgServiceServer) RecalculateTargetEmission(ctx context.Context, req *MsgServiceRecalculateTargetEmissionRequest) (*MsgServiceRecalculateTargetEmissionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecalculateTargetEmission not implemented")
}

func RegisterMsgServiceServer(s grpc1.Server, srv MsgServiceServer) {
	s.RegisterService(&_MsgService_serviceDesc, srv)
}

func _MsgService_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgServiceUpdateParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServiceServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/mint.v2.MsgService/UpdateParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServiceServer).UpdateParams(ctx, req.(*MsgServiceUpdateParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MsgService_RecalculateTargetEmission_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgServiceRecalculateTargetEmissionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServiceServer).RecalculateTargetEmission(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/mint.v2.MsgService/RecalculateTargetEmission",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServiceServer).RecalculateTargetEmission(ctx, req.(*MsgServiceRecalculateTargetEmissionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _MsgService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "mint.v2.MsgService",
	HandlerType: (*MsgServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateParams",
			Handler:    _MsgService_UpdateParams_Handler,
		},
		{
			MethodName: "RecalculateTargetEmission",
			Handler:    _MsgService_RecalculateTargetEmission_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "mint/v2/tx.proto",
}

func (m *MsgServiceUpdateParamsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgServiceUpdateParamsRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgServiceUpdateParamsRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.RecalculateTargetEmission {
		i--
		if m.RecalculateTargetEmission {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x18
	}
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Sender) > 0 {
		i -= len(m.Sender)
		copy(dAtA[i:], m.Sender)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Sender)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgServiceUpdateParamsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgServiceUpdateParamsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgServiceUpdateParamsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgServiceRecalculateTargetEmissionRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgServiceRecalculateTargetEmissionRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgServiceRecalculateTargetEmissionRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Sender) > 0 {
		i -= len(m.Sender)
		copy(dAtA[i:], m.Sender)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Sender)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgServiceRecalculateTargetEmissionResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgServiceRecalculateTargetEmissionResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgServiceRecalculateTargetEmissionResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgServiceUpdateParamsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Sender)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Params.Size()
	n += 1 + l + sovTx(uint64(l))
	if m.RecalculateTargetEmission {
		n += 2
	}
	return n
}

func (m *MsgServiceUpdateParamsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgServiceRecalculateTargetEmissionRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Sender)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgServiceRecalculateTargetEmissionResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgServiceUpdateParamsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgServiceUpdateParamsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgServiceUpdateParamsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sender", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sender = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecalculateTargetEmission", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.RecalculateTargetEmission = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgServiceUpdateParamsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgServiceUpdateParamsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgServiceUpdateParamsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgServiceRecalculateTargetEmissionRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgServiceRecalculateTargetEmissionRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgServiceRecalculateTargetEmissionRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sender", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sender = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgServiceRecalculateTargetEmissionResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgServiceRecalculateTargetEmissionResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgServiceRecalculateTargetEmissionResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
