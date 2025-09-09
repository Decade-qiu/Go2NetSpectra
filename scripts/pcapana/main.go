package main

import (
	"Go2NetSpectra/internal/engine/protocol"
	"fmt"
	"log"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

func main() {
	// 打开一个 pcap 文件（也可以换成设备名，比如 "eth0"）
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./scripts/pcapana/main.go <path_to_pcap_file>")
		os.Exit(1)
	}
	pcapFilePath := os.Args[1]
	handle, err := pcap.OpenOffline(pcapFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	i := 0
	// for packet := range packetSource.Packets() {
	// 	fmt.Printf("==== Packet %d ====\n", i+1)
	// 	for _, layer := range packet.Layers() {
	// 		fmt.Println("Layer:", layer.LayerType())
	// 	}

	// 	i++
	// 	if i >= 5 { // 只看前 5 个
	// 		break
	// 	}
	// }

	for packet := range packetSource.Packets() {
		info, err := protocol.ParsePacket(packet)
		if err != nil {
			fmt.Println("Parse error:", err)
			break
		}
		i++
		fmt.Printf("[%s] %s:%d -> %s:%d proto=%d len=%d\n",
			info.Timestamp.Format("15:04:05.000"),
			info.FiveTuple.SrcIP, info.FiveTuple.SrcPort,
			info.FiveTuple.DstIP, info.FiveTuple.DstPort,
			info.FiveTuple.Protocol, info.Length,
		)
		if i >= 5 {
			break
		}
	}
}