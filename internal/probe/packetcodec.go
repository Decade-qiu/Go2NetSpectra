package probe

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/model"
	"errors"
	"fmt"
	"net"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errNilPacketInfo  = errors.New("nil packet info")
	errNilProtoPacket = errors.New("nil proto packet")
	errNilFiveTuple   = errors.New("nil five tuple")
)

func newProtoPacket(packetInfo *model.PacketInfo) (*v1.PacketInfo, error) {
	if packetInfo == nil {
		return nil, errNilPacketInfo
	}

	return &v1.PacketInfo{
		Timestamp: timestamppb.New(packetInfo.Timestamp),
		FiveTuple: &v1.FiveTuple{
			SrcIp:    append([]byte(nil), packetInfo.FiveTuple.SrcIP...),
			DstIp:    append([]byte(nil), packetInfo.FiveTuple.DstIP...),
			SrcPort:  uint32(packetInfo.FiveTuple.SrcPort),
			DstPort:  uint32(packetInfo.FiveTuple.DstPort),
			Protocol: uint32(packetInfo.FiveTuple.Protocol),
		},
		Length: uint64(packetInfo.Length),
	}, nil
}

// PacketInfoToProto converts PacketInfo into the protobuf transport shape.
func PacketInfoToProto(packetInfo *model.PacketInfo) (*v1.PacketInfo, error) {
	return newProtoPacket(packetInfo)
}

// MarshalPacketInfo encodes PacketInfo into protobuf bytes.
func MarshalPacketInfo(dst []byte, packetInfo *model.PacketInfo) ([]byte, error) {
	protoPacket, err := newProtoPacket(packetInfo)
	if err != nil {
		return nil, err
	}

	if dst == nil {
		dst = make([]byte, 0, proto.Size(protoPacket))
	}

	data, err := proto.MarshalOptions{}.MarshalAppend(dst[:0], protoPacket)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal packet info: %w", err)
	}

	return data, nil
}

// PacketInfoFromProto converts the protobuf transport packet into PacketInfo.
func PacketInfoFromProto(packet *v1.PacketInfo) (model.PacketInfo, error) {
	if packet == nil {
		return model.PacketInfo{}, errNilProtoPacket
	}
	if packet.FiveTuple == nil {
		return model.PacketInfo{}, errNilFiveTuple
	}

	return model.PacketInfo{
		Timestamp: packet.Timestamp.AsTime(),
		Length:    int(packet.Length),
		FiveTuple: model.FiveTuple{
			SrcIP:    append(net.IP(nil), packet.FiveTuple.SrcIp...),
			DstIP:    append(net.IP(nil), packet.FiveTuple.DstIp...),
			SrcPort:  uint16(packet.FiveTuple.SrcPort),
			DstPort:  uint16(packet.FiveTuple.DstPort),
			Protocol: uint8(packet.FiveTuple.Protocol),
		},
	}, nil
}

// UnmarshalPacketInfo decodes protobuf bytes into PacketInfo.
func UnmarshalPacketInfo(data []byte) (model.PacketInfo, error) {
	var packet v1.PacketInfo
	if err := proto.Unmarshal(data, &packet); err != nil {
		return model.PacketInfo{}, fmt.Errorf("failed to unmarshal packet info: %w", err)
	}

	return PacketInfoFromProto(&packet)
}
