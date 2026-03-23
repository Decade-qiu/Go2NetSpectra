package protocol

import (
	"fmt"

	"Go2NetSpectra/internal/model"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// ParsePacket uses gopacket to decode a raw packet and extract key information.
func ParsePacket(packet gopacket.Packet) (*model.PacketInfo, error) {
	info := &model.PacketInfo{}
	if err := ParsePacketInto(packet, info); err != nil {
		return nil, err
	}

	return info, nil
}

// ParsePacketInto decodes a raw packet into a caller-provided PacketInfo.
func ParsePacketInto(packet gopacket.Packet, info *model.PacketInfo) error {
	if info == nil {
		return fmt.Errorf("nil packet info")
	}

	*info = model.PacketInfo{}

	if meta := packet.Metadata(); meta != nil {
		info.Timestamp = meta.Timestamp
		info.Length = meta.Length
	}

	var fiveTuple model.FiveTuple

	// Get IP layer
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		fiveTuple.SrcIP = ip.SrcIP
		fiveTuple.DstIP = ip.DstIP
		fiveTuple.Protocol = uint8(ip.Protocol)
	} else if ipLayer := packet.Layer(layers.LayerTypeIPv6); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv6)
		fiveTuple.SrcIP = ip.SrcIP
		fiveTuple.DstIP = ip.DstIP
		fiveTuple.Protocol = uint8(ip.NextHeader)
	} else {
		return fmt.Errorf("not an IP packet")
	}

	// Get transport layer
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		fiveTuple.SrcPort = uint16(tcp.SrcPort)
		fiveTuple.DstPort = uint16(tcp.DstPort)
	} else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		fiveTuple.SrcPort = uint16(udp.SrcPort)
		fiveTuple.DstPort = uint16(udp.DstPort)
	}
	// For other protocols like ICMP, the ports will be 0, which is correct.

	info.FiveTuple = fiveTuple

	return nil
}
