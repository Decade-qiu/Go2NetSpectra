package protocol

import (
	"Go2NetSpectra/internal/core/model"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"time"
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
	var ipLayer *layers.IPv4
	var tcpLayer *layers.TCP
	var udpLayer *layers.UDP

	// Get IPv4 layer
	if l := packet.Layer(layers.LayerTypeIPv4); l != nil {
		ipLayer = l.(*layers.IPv4)
		fiveTuple.SrcIP = ipLayer.SrcIP
		fiveTuple.DstIP = ipLayer.DstIP
		fiveTuple.Protocol = uint8(ipLayer.Protocol)
	} else {
		// Handle IPv6 if necessary, for now we skip non-IPv4
		return nil, fmt.Errorf("not an IPv4 packet")
	}

	// Get TCP layer
	if l := packet.Layer(layers.LayerTypeTCP); l != nil {
		tcpLayer = l.(*layers.TCP)
		fiveTuple.SrcPort = uint16(tcpLayer.SrcPort)
		fiveTuple.DstPort = uint16(tcpLayer.DstPort)
	} else if l := packet.Layer(layers.LayerTypeUDP); l != nil {
		// Get UDP layer
		udpLayer = l.(*layers.UDP)
		fiveTuple.SrcPort = uint16(udpLayer.SrcPort)
		fiveTuple.DstPort = uint16(udpLayer.DstPort)
	} else {
		// Not a TCP or UDP packet, we can choose to ignore or handle it
		return nil, fmt.Errorf("not a TCP or UDP packet")
	}

	info.FiveTuple = fiveTuple

	return info, nil
}
