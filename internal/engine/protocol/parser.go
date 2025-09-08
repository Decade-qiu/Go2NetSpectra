package protocol

import (
	"Go2NetSpectra/internal/model"
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// ParsePacket uses gopacket to decode a raw packet and extract key information.
func ParsePacket(data []byte) (*model.PacketInfo, error) {
	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)

	info := &model.PacketInfo{
		Timestamp: time.Now(), // Default to now, will be overwritten by packet metadata if available
		Length:    len(data),
	}

	if meta := packet.Metadata(); meta != nil {
		info.Timestamp = meta.Timestamp
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
		return nil, fmt.Errorf("not an IP packet")
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

	return info, nil
}
