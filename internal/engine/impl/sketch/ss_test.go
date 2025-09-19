package sketch

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/sketch/statistic"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/pkg/pcap"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"testing"
)

func TestSuperSpread(t *testing.T) {
	pcapFilePath := "../../../../test/data/caida.pcap"
	pcapReader, err := pcap.NewReader(pcapFilePath)
	if err != nil {
		log.Fatalf("Failed to open pcap file: %v", err)
	}
	defer pcapReader.Close()
	log.Printf("Reading packets from '%s'...", pcapFilePath)

	packetChannel := make(chan *v1.PacketInfo, 10000)

	SpreadThreshold := uint32(750)

	// Initialize SuperSpread sketch
	cfg := config.SketchTaskDef{
		Name:            "per_src_flow",
		SktType:         1,
		FlowFields:      []string{"DstIP"},
		ElementFields:   []string{"SrcIP"},
		Width:           1 << 13,
		Depth:           2,
		SizeThereshold:  0,
		CountThereshold: 750,
		M: 128,
		Base: 0.5,
		Size: 5,
		B: 1.08,
	}

	task := New(cfg)

	// Ground truth (map-based)
	spreadMap := make(map[string]map[string]bool)

	var wg sync.WaitGroup

	wg.Go(func() {
		for pbPacket := range packetChannel {
			info := &model.PacketInfo{
				Timestamp: pbPacket.Timestamp.AsTime(),
				Length:    int(pbPacket.Length),
				FiveTuple: model.FiveTuple{
					SrcIP:    net.IP(pbPacket.FiveTuple.SrcIp).To16(),
					DstIP:    net.IP(pbPacket.FiveTuple.DstIp).To16(),
					SrcPort:  uint16(pbPacket.FiveTuple.SrcPort),
					DstPort:  uint16(pbPacket.FiveTuple.DstPort),
					Protocol: uint8(pbPacket.FiveTuple.Protocol),
				},
			}

			task.ProcessPacket(info)

			key := info.FiveTuple.DstIP.String()
			elem := fmt.Sprintf("%s",
				info.FiveTuple.SrcIP.String())
			if _, exists := spreadMap[key]; !exists {
				spreadMap[key] = make(map[string]bool)
			}
			spreadMap[key][elem] = true
		}
	})

	log.Println("Start processing...")
	pcapReader.ReadPackets(packetChannel)
	close(packetChannel)
	wg.Wait()
	log.Println("Finished reading all packets.")

	// ---------------------------
	// Per-flow Spread Error calculation
	// ---------------------------
	file, err := os.Create("spread.txt")
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	spreadRelErrSum := 0.0

	for key, elemSet := range spreadMap {
		actualSpread := len(elemSet) // Ground truth spread
		flow := net.ParseIP(key)
		estimatedSpread := task.Query(flow)

		// Relative Error
		spreadRE := float64(estimatedSpread)-float64(actualSpread) / float64(actualSpread)
		if spreadRE < 0 {
			spreadRE = -spreadRE
		}
		spreadRelErrSum += spreadRE

		_, err := fmt.Fprintf(writer, "%s %d %d\n",
			key, actualSpread, estimatedSpread)
		if err != nil {
			log.Fatalf("Failed to write: %v", err)
		}
	}

	avgSpreadRE := spreadRelErrSum / float64(len(spreadMap))
	log.Printf("Per-flow Avg Relative Error (Spread): %.4f", avgSpreadRE)

	// ---------------------------
	// Superspreader Detection
	// ---------------------------
	res := task.Snapshot().(statistic.HeavyRecord).Count

	// Ground truth Superspreaders
	trueSuperspreaders := make(map[string]int)
	for key, elemSet := range spreadMap {
		if uint32(len(elemSet)) >= SpreadThreshold {
			trueSuperspreaders[key] = len(elemSet)
		}
	}

	// Detected Superspreaders
	detectedSuperspreaders := make(map[string]uint32)
	for _, record := range res {
		key := net.IP(record.Flow).String()
		detectedSuperspreaders[key] = record.Count
	}

	// Evaluate
	spreadMRE, spreadPrec, spreadRec, spreadF1, tp, fp, fn :=
		evaluateHeavyHitters(detectedSuperspreaders, trueSuperspreaders)

	log.Printf("[Superspread] TP=%d FP=%d FN=%d", tp, fp, fn)
	log.Printf("[Superspread] MRE=%.4f Precision=%.4f Recall=%.4f F1=%.4f",
		spreadMRE, spreadPrec, spreadRec, spreadF1)
}
