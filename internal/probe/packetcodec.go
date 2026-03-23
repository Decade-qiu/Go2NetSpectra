package probe

import (
	"Go2NetSpectra/internal/model"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	v1 "Go2NetSpectra/api/gen/thrift/v1"

	thrift "github.com/apache/thrift/lib/go/thrift"
)

var (
	errNilPacketInfo   = errors.New("nil packet info")
	errNilThriftPacket = errors.New("nil thrift packet")
	errNilFiveTuple    = errors.New("nil five tuple")
)

var packetSerializerPool = sync.Pool{
	New: func() any {
		return thrift.NewTSerializer()
	},
}

var packetDeserializerPool = sync.Pool{
	New: func() any {
		return thrift.NewTDeserializer()
	},
}

func packetInfoToThrift(packetInfo *model.PacketInfo) (*v1.PacketInfo, error) {
	if packetInfo == nil {
		return nil, errNilPacketInfo
	}

	return &v1.PacketInfo{
		TimestampUnixNano: packetInfo.Timestamp.UnixNano(),
		FiveTuple: &v1.FiveTuple{
			SrcIP:    append([]byte(nil), packetInfo.FiveTuple.SrcIP...),
			DstIP:    append([]byte(nil), packetInfo.FiveTuple.DstIP...),
			SrcPort:  int32(packetInfo.FiveTuple.SrcPort),
			DstPort:  int32(packetInfo.FiveTuple.DstPort),
			Protocol: int32(packetInfo.FiveTuple.Protocol),
		},
		Length: int64(packetInfo.Length),
	}, nil
}

// MarshalPacketInfo encodes PacketInfo into Thrift bytes.
func MarshalPacketInfo(dst []byte, packetInfo *model.PacketInfo) ([]byte, error) {
	thriftPacket, err := packetInfoToThrift(packetInfo)
	if err != nil {
		return nil, err
	}

	serializer := packetSerializerPool.Get().(*thrift.TSerializer)
	defer packetSerializerPool.Put(serializer)

	data, err := serializer.Write(context.Background(), thriftPacket)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal packet info: %w", err)
	}

	if dst == nil {
		return data, nil
	}

	return append(dst[:0], data...), nil
}

func packetInfoFromThrift(packet *v1.PacketInfo) (model.PacketInfo, error) {
	if packet == nil {
		return model.PacketInfo{}, errNilThriftPacket
	}
	if packet.FiveTuple == nil {
		return model.PacketInfo{}, errNilFiveTuple
	}

	return model.PacketInfo{
		Timestamp: time.Unix(0, packet.TimestampUnixNano),
		Length:    int(packet.Length),
		FiveTuple: model.FiveTuple{
			SrcIP:    append(net.IP(nil), packet.FiveTuple.SrcIP...),
			DstIP:    append(net.IP(nil), packet.FiveTuple.DstIP...),
			SrcPort:  uint16(packet.FiveTuple.SrcPort),
			DstPort:  uint16(packet.FiveTuple.DstPort),
			Protocol: uint8(packet.FiveTuple.Protocol),
		},
	}, nil
}

// UnmarshalPacketInfo decodes Thrift bytes into PacketInfo.
func UnmarshalPacketInfo(data []byte) (model.PacketInfo, error) {
	packet := v1.NewPacketInfo()
	deserializer := packetDeserializerPool.Get().(*thrift.TDeserializer)
	defer packetDeserializerPool.Put(deserializer)

	if err := deserializer.Read(context.Background(), packet, data); err != nil {
		return model.PacketInfo{}, fmt.Errorf("failed to unmarshal packet info: %w", err)
	}

	return packetInfoFromThrift(packet)
}
