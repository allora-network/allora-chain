// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: emissions/v3/inference.proto

package types

import (
	fmt "fmt"
	github_com_allora_network_allora_chain_math "github.com/allora-network/allora-chain/math"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
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

type RegretInformedWeight struct {
	Worker string                                          `protobuf:"bytes,1,opt,name=worker,proto3" json:"worker,omitempty"`
	Weight github_com_allora_network_allora_chain_math.Dec `protobuf:"bytes,2,opt,name=weight,proto3,customtype=github.com/allora-network/allora-chain/math.Dec" json:"weight"`
}

func (m *RegretInformedWeight) Reset()         { *m = RegretInformedWeight{} }
func (m *RegretInformedWeight) String() string { return proto.CompactTextString(m) }
func (*RegretInformedWeight) ProtoMessage()    {}
func (*RegretInformedWeight) Descriptor() ([]byte, []int) {
	return fileDescriptor_71dd7e4b5a3958bf, []int{0}
}
func (m *RegretInformedWeight) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RegretInformedWeight) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RegretInformedWeight.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *RegretInformedWeight) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RegretInformedWeight.Merge(m, src)
}
func (m *RegretInformedWeight) XXX_Size() int {
	return m.Size()
}
func (m *RegretInformedWeight) XXX_DiscardUnknown() {
	xxx_messageInfo_RegretInformedWeight.DiscardUnknown(m)
}

var xxx_messageInfo_RegretInformedWeight proto.InternalMessageInfo

func (m *RegretInformedWeight) GetWorker() string {
	if m != nil {
		return m.Worker
	}
	return ""
}

func init() {
	proto.RegisterType((*RegretInformedWeight)(nil), "emissions.v3.RegretInformedWeight")
}

func init() { proto.RegisterFile("emissions/v3/inference.proto", fileDescriptor_71dd7e4b5a3958bf) }

var fileDescriptor_71dd7e4b5a3958bf = []byte{
	// 233 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x49, 0xcd, 0xcd, 0x2c,
	0x2e, 0xce, 0xcc, 0xcf, 0x2b, 0xd6, 0x2f, 0x33, 0xd6, 0xcf, 0xcc, 0x4b, 0x4b, 0x2d, 0x4a, 0xcd,
	0x4b, 0x4e, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x81, 0xcb, 0xea, 0x95, 0x19, 0x4b,
	0x89, 0xa4, 0xe7, 0xa7, 0xe7, 0x83, 0x25, 0xf4, 0x41, 0x2c, 0x88, 0x1a, 0xa5, 0x56, 0x46, 0x2e,
	0x91, 0xa0, 0xd4, 0xf4, 0xa2, 0xd4, 0x12, 0xcf, 0xbc, 0xb4, 0xfc, 0xa2, 0xdc, 0xd4, 0x94, 0xf0,
	0xd4, 0xcc, 0xf4, 0x8c, 0x12, 0x21, 0x31, 0x2e, 0xb6, 0xf2, 0xfc, 0xa2, 0xec, 0xd4, 0x22, 0x09,
	0x46, 0x05, 0x46, 0x0d, 0xce, 0x20, 0x28, 0x4f, 0xc8, 0x9f, 0x8b, 0xad, 0x1c, 0xac, 0x42, 0x82,
	0x09, 0x24, 0xee, 0x64, 0x7e, 0xe2, 0x9e, 0x3c, 0xc3, 0xad, 0x7b, 0xf2, 0xfa, 0xe9, 0x99, 0x25,
	0x19, 0xa5, 0x49, 0x7a, 0xc9, 0xf9, 0xb9, 0xfa, 0x89, 0x39, 0x39, 0xf9, 0x45, 0x89, 0xba, 0x79,
	0xa9, 0x25, 0x20, 0x4d, 0x30, 0x6e, 0x72, 0x46, 0x62, 0x66, 0x9e, 0x7e, 0x6e, 0x62, 0x49, 0x86,
	0x9e, 0x4b, 0x6a, 0x72, 0x10, 0xd4, 0x18, 0x2b, 0x96, 0x17, 0x0b, 0xe4, 0x19, 0x9d, 0x82, 0x4e,
	0x3c, 0x92, 0x63, 0xbc, 0xf0, 0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x09, 0x8f, 0xe5, 0x18,
	0x2e, 0x3c, 0x96, 0x63, 0xb8, 0xf1, 0x58, 0x8e, 0x21, 0xca, 0x82, 0x48, 0x83, 0x2b, 0xf4, 0x11,
	0x81, 0x51, 0x52, 0x59, 0x90, 0x5a, 0x9c, 0xc4, 0x06, 0xf6, 0xa2, 0x31, 0x20, 0x00, 0x00, 0xff,
	0xff, 0x8f, 0x48, 0xf6, 0xfb, 0x26, 0x01, 0x00, 0x00,
}

func (this *RegretInformedWeight) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*RegretInformedWeight)
	if !ok {
		that2, ok := that.(RegretInformedWeight)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Worker != that1.Worker {
		return false
	}
	if !this.Weight.Equal(that1.Weight) {
		return false
	}
	return true
}
func (m *RegretInformedWeight) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RegretInformedWeight) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *RegretInformedWeight) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Weight.Size()
		i -= size
		if _, err := m.Weight.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintInference(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Worker) > 0 {
		i -= len(m.Worker)
		copy(dAtA[i:], m.Worker)
		i = encodeVarintInference(dAtA, i, uint64(len(m.Worker)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintInference(dAtA []byte, offset int, v uint64) int {
	offset -= sovInference(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *RegretInformedWeight) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Worker)
	if l > 0 {
		n += 1 + l + sovInference(uint64(l))
	}
	l = m.Weight.Size()
	n += 1 + l + sovInference(uint64(l))
	return n
}

func sovInference(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozInference(x uint64) (n int) {
	return sovInference(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *RegretInformedWeight) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowInference
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
			return fmt.Errorf("proto: RegretInformedWeight: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RegretInformedWeight: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Worker", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInference
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
				return ErrInvalidLengthInference
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthInference
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Worker = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Weight", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowInference
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
				return ErrInvalidLengthInference
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthInference
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Weight.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipInference(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthInference
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
func skipInference(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowInference
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
					return 0, ErrIntOverflowInference
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
					return 0, ErrIntOverflowInference
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
				return 0, ErrInvalidLengthInference
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupInference
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthInference
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthInference        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowInference          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupInference = fmt.Errorf("proto: unexpected end of group")
)
