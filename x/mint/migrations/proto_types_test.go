package migrations_test

import (
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	cosmossdk_io_math "cosmossdk.io/math"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
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

// OldParams defines the parameters for the x/mint module.
type OldParams struct {
	// type of coin to mint
	MintDenom string `protobuf:"bytes,1,opt,name=mint_denom,json=mintDenom,proto3" json:"mint_denom,omitempty"`
	// maximum total supply of the coin
	MaxSupply cosmossdk_io_math.Int `protobuf:"bytes,2,opt,name=max_supply,json=maxSupply,proto3,customtype=cosmossdk.io/math.Int" json:"max_supply"`
	// ecosystem treasury fraction ideally emitted per unit time
	FEmission cosmossdk_io_math.LegacyDec `protobuf:"bytes,3,opt,name=f_emission,json=fEmission,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"f_emission"`
	// one month exponential moving average smoothing factor, alpha_e in the paper
	OneMonthSmoothingDegree cosmossdk_io_math.LegacyDec `protobuf:"bytes,4,opt,name=one_month_smoothing_degree,json=oneMonthSmoothingDegree,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"one_month_smoothing_degree"`
	// percentage of the total supply is reserved and locked in the ecosystem treasury
	EcosystemTreasuryPercentOfTotalSupply cosmossdk_io_math.LegacyDec `protobuf:"bytes,5,opt,name=ecosystem_treasury_percent_of_total_supply,json=ecosystemTreasuryPercentOfTotalSupply,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"ecosystem_treasury_percent_of_total_supply"`
	// percentage of the total supply that is unlocked and usable in the foundation treasury
	FoundationTreasuryPercentOfTotalSupply cosmossdk_io_math.LegacyDec `protobuf:"bytes,6,opt,name=foundation_treasury_percent_of_total_supply,json=foundationTreasuryPercentOfTotalSupply,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"foundation_treasury_percent_of_total_supply"`
	// percentage of the total supply that is unlocked and usable by partipicants at the genesis
	ParticipantsPercentOfTotalSupply cosmossdk_io_math.LegacyDec `protobuf:"bytes,7,opt,name=participants_percent_of_total_supply,json=participantsPercentOfTotalSupply,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"participants_percent_of_total_supply"`
	// percentage of the total supply that is locked in the investors bucket at the genesis
	InvestorsPercentOfTotalSupply cosmossdk_io_math.LegacyDec `protobuf:"bytes,8,opt,name=investors_percent_of_total_supply,json=investorsPercentOfTotalSupply,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"investors_percent_of_total_supply"`
	// percentage of the total supply that is locked in the team bucket at the genesis
	TeamPercentOfTotalSupply cosmossdk_io_math.LegacyDec `protobuf:"bytes,9,opt,name=team_percent_of_total_supply,json=teamPercentOfTotalSupply,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"team_percent_of_total_supply"`
	// The capped max monthly percentage yield (like %APY)
	MaximumMonthlyPercentageYield cosmossdk_io_math.LegacyDec `protobuf:"bytes,10,opt,name=maximum_monthly_percentage_yield,json=maximumMonthlyPercentageYield,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"maximum_monthly_percentage_yield"`
}

func (m *OldParams) Reset()         { *m = OldParams{} }
func (m *OldParams) String() string { return proto.CompactTextString(m) }
func (*OldParams) ProtoMessage()    {}
func (*OldParams) Descriptor() ([]byte, []int) {
	return fileDescriptor_010015e812760429, []int{0}
}
func (m *OldParams) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *OldParams) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *OldParams) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *OldParams) XXX_Size() int {
	return m.Size()
}
func (m *OldParams) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *OldParams) GetMintDenom() string {
	if m != nil {
		return m.MintDenom
	}
	return ""
}

func init() {
	proto.RegisterType((*OldParams)(nil), "mint.v1beta1.Params")
}

func init() { proto.RegisterFile("mint/v1beta1/types.proto", fileDescriptor_010015e812760429) }

var fileDescriptor_010015e812760429 = []byte{
	// 551 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x94, 0x3f, 0x6f, 0xd3, 0x40,
	0x18, 0xc6, 0x73, 0xfc, 0x29, 0xe4, 0xc4, 0x00, 0x16, 0x08, 0x13, 0xa8, 0x1b, 0x10, 0x20, 0x14,
	0x94, 0x98, 0xaa, 0x1b, 0x62, 0xaa, 0xc2, 0x50, 0x89, 0xa8, 0x51, 0xdb, 0x05, 0x18, 0x4e, 0x17,
	0xe7, 0xe2, 0x9c, 0xea, 0xbb, 0xd7, 0xf2, 0xbd, 0x29, 0xb1, 0xc4, 0x0a, 0x03, 0x62, 0x60, 0x43,
	0x7c, 0x03, 0xc6, 0x0e, 0xfd, 0x0a, 0x48, 0x1d, 0xab, 0x4e, 0x88, 0xa1, 0x42, 0xc9, 0xd0, 0xaf,
	0x81, 0x6c, 0x5f, 0x03, 0x12, 0x09, 0x2c, 0x5e, 0x2c, 0xfb, 0x7d, 0xac, 0xdf, 0xef, 0xd1, 0xc9,
	0x7e, 0xa9, 0xab, 0xa4, 0x46, 0x7f, 0x6f, 0xb5, 0x27, 0x90, 0xaf, 0xfa, 0x98, 0xc6, 0xc2, 0xb4,
	0xe2, 0x04, 0x10, 0x9c, 0x2b, 0x59, 0xd2, 0xb2, 0x49, 0xed, 0x7a, 0x08, 0x21, 0xe4, 0x81, 0x9f,
	0xdd, 0x15, 0xef, 0xd4, 0x6e, 0x05, 0x60, 0x14, 0x18, 0x56, 0x04, 0xc5, 0x83, 0x8d, 0xae, 0x71,
	0x25, 0x35, 0xf8, 0xf9, 0xb5, 0x18, 0xdd, 0xfb, 0x56, 0xa5, 0x4b, 0x5d, 0x9e, 0x70, 0x65, 0x9c,
	0x65, 0x4a, 0x33, 0x3c, 0xeb, 0x0b, 0x0d, 0xca, 0x25, 0x75, 0xf2, 0xa8, 0xba, 0x55, 0xcd, 0x26,
	0xed, 0x6c, 0xe0, 0x6c, 0x52, 0xaa, 0xf8, 0x98, 0x99, 0x51, 0x1c, 0x47, 0xa9, 0x7b, 0x2e, 0x8b,
	0xd7, 0x9f, 0x1c, 0x9e, 0xac, 0x54, 0x7e, 0x9c, 0xac, 0xdc, 0x28, 0x34, 0xa6, 0xbf, 0xdb, 0x92,
	0xe0, 0x2b, 0x8e, 0xc3, 0xd6, 0x86, 0xc6, 0xe3, 0x83, 0x26, 0xb5, 0xfe, 0x0d, 0x8d, 0x5f, 0x4f,
	0xf7, 0x1b, 0x64, 0xab, 0xaa, 0xf8, 0x78, 0x3b, 0x47, 0x38, 0xaf, 0x29, 0x1d, 0x30, 0xa1, 0xa4,
	0x31, 0x12, 0xb4, 0x7b, 0x3e, 0x07, 0x3e, 0xb3, 0xc0, 0xdb, 0x7f, 0x03, 0x5f, 0x88, 0x90, 0x07,
	0x69, 0x5b, 0x04, 0xc7, 0x07, 0xcd, 0xab, 0x16, 0x3b, 0x9b, 0x59, 0xf8, 0xe0, 0xb9, 0xc5, 0x39,
	0x29, 0xad, 0x81, 0x16, 0x4c, 0x81, 0xc6, 0x21, 0x33, 0x0a, 0x00, 0x87, 0x52, 0x87, 0xac, 0x2f,
	0xc2, 0x44, 0x08, 0xf7, 0x42, 0x09, 0xb2, 0x9b, 0xa0, 0x45, 0x27, 0xc3, 0x6f, 0x9f, 0xd1, 0xdb,
	0x39, 0xdc, 0xf9, 0x4c, 0x68, 0x43, 0x04, 0x60, 0x52, 0x83, 0x42, 0x31, 0x4c, 0x04, 0x37, 0xa3,
	0x24, 0x65, 0xb1, 0x48, 0x02, 0xa1, 0x91, 0xc1, 0x80, 0x21, 0x20, 0x8f, 0xce, 0x4e, 0xf2, 0x62,
	0x09, 0x5d, 0x1e, 0xcc, 0x7c, 0x3b, 0x56, 0xd7, 0x2d, 0x6c, 0x9b, 0x83, 0x9d, 0xcc, 0x65, 0x4f,
	0xfc, 0x0b, 0xa1, 0x8f, 0x07, 0x30, 0xd2, 0x7d, 0x8e, 0x12, 0xf4, 0xff, 0xab, 0x2d, 0x95, 0x50,
	0xed, 0xe1, 0x6f, 0xe1, 0x3f, 0xbb, 0x7d, 0x24, 0xf4, 0x7e, 0xcc, 0x13, 0x94, 0x81, 0x8c, 0xb9,
	0x46, 0xb3, 0xb0, 0xd4, 0xa5, 0x12, 0x4a, 0xd5, 0xff, 0x34, 0xcd, 0xad, 0xf3, 0x9e, 0xd0, 0xbb,
	0x52, 0xef, 0x09, 0x83, 0x90, 0x2c, 0xee, 0x72, 0xb9, 0x84, 0x2e, 0xcb, 0x33, 0xcd, 0xdc, 0x22,
	0x6f, 0xe9, 0x1d, 0x14, 0x5c, 0x2d, 0xac, 0x50, 0x2d, 0xa1, 0x82, 0x9b, 0x19, 0xe6, 0xda, 0xdf,
	0x11, 0x5a, 0x57, 0x7c, 0x2c, 0xd5, 0x48, 0x15, 0xff, 0x52, 0x34, 0xfb, 0x5a, 0x78, 0x28, 0x58,
	0x2a, 0x45, 0xd4, 0x77, 0x69, 0x19, 0xa7, 0x60, 0x2d, 0x9d, 0x42, 0xd2, 0x9d, 0x39, 0x5e, 0x66,
	0x8a, 0xa7, 0x2b, 0x1f, 0x4e, 0xf7, 0x1b, 0x35, 0x1e, 0x45, 0x90, 0xf0, 0x66, 0x30, 0xe4, 0x52,
	0xfb, 0x63, 0x3f, 0x5f, 0x93, 0xc5, 0xf2, 0x5a, 0xef, 0x1c, 0x4e, 0x3c, 0x72, 0x34, 0xf1, 0xc8,
	0xcf, 0x89, 0x47, 0x3e, 0x4d, 0xbd, 0xca, 0xd1, 0xd4, 0xab, 0x7c, 0x9f, 0x7a, 0x95, 0x57, 0x6b,
	0xa1, 0xc4, 0xe1, 0xa8, 0xd7, 0x0a, 0x40, 0xf9, 0x16, 0xa0, 0x05, 0xbe, 0x81, 0x64, 0xd7, 0x9f,
	0xc7, 0xcb, 0xd7, 0x6d, 0x6f, 0x29, 0xdf, 0x8e, 0x6b, 0xbf, 0x02, 0x00, 0x00, 0xff, 0xff, 0x8e,
	0x93, 0xc4, 0x89, 0x8b, 0x05, 0x00, 0x00,
}

func (m *OldParams) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *OldParams) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *OldParams) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.MaximumMonthlyPercentageYield.Size()
		i -= size
		if _, err := m.MaximumMonthlyPercentageYield.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x52
	{
		size := m.TeamPercentOfTotalSupply.Size()
		i -= size
		if _, err := m.TeamPercentOfTotalSupply.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x4a
	{
		size := m.InvestorsPercentOfTotalSupply.Size()
		i -= size
		if _, err := m.InvestorsPercentOfTotalSupply.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x42
	{
		size := m.ParticipantsPercentOfTotalSupply.Size()
		i -= size
		if _, err := m.ParticipantsPercentOfTotalSupply.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x3a
	{
		size := m.FoundationTreasuryPercentOfTotalSupply.Size()
		i -= size
		if _, err := m.FoundationTreasuryPercentOfTotalSupply.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x32
	{
		size := m.EcosystemTreasuryPercentOfTotalSupply.Size()
		i -= size
		if _, err := m.EcosystemTreasuryPercentOfTotalSupply.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	{
		size := m.OneMonthSmoothingDegree.Size()
		i -= size
		if _, err := m.OneMonthSmoothingDegree.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
	{
		size := m.FEmission.Size()
		i -= size
		if _, err := m.FEmission.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	{
		size := m.MaxSupply.Size()
		i -= size
		if _, err := m.MaxSupply.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.MintDenom) > 0 {
		i -= len(m.MintDenom)
		copy(dAtA[i:], m.MintDenom)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.MintDenom)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTypes(dAtA []byte, offset int, v uint64) int {
	offset -= sovTypes(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *OldParams) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.MintDenom)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	l = m.MaxSupply.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = m.FEmission.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = m.OneMonthSmoothingDegree.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = m.EcosystemTreasuryPercentOfTotalSupply.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = m.FoundationTreasuryPercentOfTotalSupply.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = m.ParticipantsPercentOfTotalSupply.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = m.InvestorsPercentOfTotalSupply.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = m.TeamPercentOfTotalSupply.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = m.MaximumMonthlyPercentageYield.Size()
	n += 1 + l + sovTypes(uint64(l))
	return n
}

func sovTypes(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func (m *OldParams) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypes
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
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MintDenom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MintDenom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxSupply", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MaxSupply.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FEmission", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.FEmission.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OneMonthSmoothingDegree", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.OneMonthSmoothingDegree.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field EcosystemTreasuryPercentOfTotalSupply", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.EcosystemTreasuryPercentOfTotalSupply.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FoundationTreasuryPercentOfTotalSupply", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.FoundationTreasuryPercentOfTotalSupply.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ParticipantsPercentOfTotalSupply", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ParticipantsPercentOfTotalSupply.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InvestorsPercentOfTotalSupply", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.InvestorsPercentOfTotalSupply.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TeamPercentOfTotalSupply", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.TeamPercentOfTotalSupply.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaximumMonthlyPercentageYield", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MaximumMonthlyPercentageYield.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTypes(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTypes
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
func skipTypes(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTypes
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
					return 0, ErrIntOverflowTypes
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
					return 0, ErrIntOverflowTypes
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
				return 0, ErrInvalidLengthTypes
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTypes
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTypes
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTypes        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTypes          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTypes = fmt.Errorf("proto: unexpected end of group")
)
