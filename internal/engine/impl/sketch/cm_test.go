package sketch

import (
	v1 "Go2NetSpectra/api/gen/v1"
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

func TestCountMin(t *testing.T) {
	pcapFilePath := "../../../../test/data/caida.pcap"
	pcapReader, err := pcap.NewReader(pcapFilePath)
	if err != nil {
		log.Fatalf("Failed to open pcap file: %v", err)
	}
	defer pcapReader.Close()
	log.Printf("Reading packets from '%s'...", pcapFilePath)

	var packetChannel chan *v1.PacketInfo = make(chan *v1.PacketInfo, 10000)

	thereshold := uint32(8096)

	// sketch
	task := New("per_src_flow", []string{"SrcIP"}, []string{"DstIP", "SrcPort", "DstPort", "Protocol"}, 1<<15, 3, thereshold)
	// map
	hashmap := make(map[string]int)

	var wg sync.WaitGroup

	wg.Go(func() {
		for pbPacket := range packetChannel {
			info := &model.PacketInfo{
				Timestamp: pbPacket.Timestamp.AsTime(),
				Length:    int(pbPacket.Length),
				FiveTuple: model.FiveTuple{
					SrcIP:    net.IP(pbPacket.FiveTuple.SrcIp),
					DstIP:    net.IP(pbPacket.FiveTuple.DstIp),
					SrcPort:  uint16(pbPacket.FiveTuple.SrcPort),
					DstPort:  uint16(pbPacket.FiveTuple.DstPort),
					Protocol: uint8(pbPacket.FiveTuple.Protocol),
				},
			}

			task.ProcessPacket(info)

			key := info.FiveTuple.SrcIP.String()
			hashmap[key] += 1
		}
	})

	log.Println("Start processing...")
	pcapReader.ReadPackets(packetChannel)
	close(packetChannel)
	wg.Wait()
	log.Println("Finished reading all packets from pcap file.")

	// per-flow size
	file, err := os.Create("flow.txt")
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	if err != nil {
		log.Fatalf("Failed to create writer: %v", err)
	}
	defer writer.Flush()
	relativeErrorSum := 0.0
	for key, actualCount := range hashmap {
		flow := net.ParseIP(key).To16()
		estimatedCount := int(task.Query(flow))
		relativeError := float64(estimatedCount-actualCount) / float64(actualCount)
		if relativeError < 0 {
			relativeError = -relativeError
		}
		relativeErrorSum += relativeError

		_, err := writer.WriteString(key + " " + fmt.Sprintf("%d", actualCount) + " " + fmt.Sprintf("%d", estimatedCount) + "\n")
		if err != nil {
			log.Fatalf("Failed to write to output file: %v", err)
		}
	}
	avgRelativeError := relativeErrorSum / float64(len(hashmap))
	log.Printf("Per-flow Average Relative Error: %.4f\n", avgRelativeError)

	// heavy hitters
	res := task.Snapshot().([]statistic.HeavyRecord)

	// Calculate MRE and F1-Score for heavy hitters
	heavyHitterMRESum := 0.0
	truePositives := 0
	falsePositives := 0
	falseNegatives := 0

	// Create a map of true heavy hitters for efficient lookup
	trueHeavyHitters := make(map[string]int)
	for key, count := range hashmap {
		if uint32(count) >= thereshold {
			trueHeavyHitters[key] = count
		}
	}

	// Build a set of detected heavy hitters for FN calculation
	detectedHH := make(map[string]uint32)
	for _, record := range res {
		key := net.IP(record.Flow).String()
		detectedHH[key] = record.Count

		if actualCount, isTrue := trueHeavyHitters[key]; isTrue {
			// True Positive
			truePositives++
			estimatedCount := int(record.Count)
			relativeError := float64(estimatedCount-actualCount) / float64(actualCount)
			if relativeError < 0 {
				relativeError = -relativeError
			}
			heavyHitterMRESum += relativeError
		} else {
			// False Positive
			falsePositives++
		}
	}

	// False Negatives = true HHs not detected
	for key := range trueHeavyHitters {
		if _, found := detectedHH[key]; !found {
			falseNegatives++
		}
	}

	// MRE for Heavy Hitters
	var heavyHitterMRE float64
	if truePositives > 0 {
		heavyHitterMRE = heavyHitterMRESum / float64(truePositives)
	}
	log.Printf("Heavy Hitters MRE: %.4f\n", heavyHitterMRE)

	// Precision, Recall, F1
	precision := 0.0
	recall := 0.0
	if (truePositives + falsePositives) > 0 {
		precision = float64(truePositives) / float64(truePositives+falsePositives)
	}
	if (truePositives + falseNegatives) > 0 {
		recall = float64(truePositives) / float64(truePositives+falseNegatives)
	}

	f1Score := 0.0
	if (precision + recall) > 0 {
		f1Score = 2 * (precision * recall) / (precision + recall)
	}

	log.Printf("True Positives: %d", truePositives)
	log.Printf("False Positives: %d", falsePositives)
	log.Printf("True Negatives: %d", len(hashmap)-truePositives)
	log.Printf("False Negatives: %d", falseNegatives)

	log.Printf("Heavy Hitters Precision: %.4f", precision)
	log.Printf("Heavy Hitters Recall: %.4f", recall)
	log.Printf("Heavy Hitters F1-Score: %.4f\n", f1Score)
}
