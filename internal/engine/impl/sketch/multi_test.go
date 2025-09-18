package sketch

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/engine/impl/exact"
	"Go2NetSpectra/internal/engine/impl/sketch/statistic"
	"Go2NetSpectra/internal/model"
	"Go2NetSpectra/pkg/pcap"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"
)

func TestMultiProcess(t *testing.T) {
	pcapFilePath := "../../../../test/data/caida.pcap"
	pcapReader, err := pcap.NewReader(pcapFilePath)
	if err != nil {
		log.Fatalf("Failed to open pcap file: %v", err)
	}
	defer pcapReader.Close()
	log.Printf("Reading packets from '%s'...", pcapFilePath)

	packetChannel := make(chan *v1.PacketInfo, 10000)

	CountThreshold := uint32(4096)
	SizeThreshold := uint32(4096 * 1024)

	// Initialize CountMin sketch
	// task := New("per_src_flow", []string{"SrcIP"}, []string{"DstIP", "SrcPort", "DstPort", "Protocol"}, 1<<15, 2, SizeThreshold, CountThreshold)

	task := exact.New("exact_per_src", []string{"SrcIP"}, 64)

	// Ground truth (map-based)
	countMap := make(map[string]int)
	sizeMap := make(map[string]int)
	var mu sync.Mutex // protect maps

	numWorkers := 28 // N 个并发消费者
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
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

				// Insert into sketch
				task.ProcessPacket(info)

				// Update ground truth maps safely
				key := info.FiveTuple.SrcIP.String()
				mu.Lock()
				countMap[key]++
				sizeMap[key] += info.Length
				mu.Unlock()
			}
		}()
	}

	log.Println("Start processing...")
	pcapReader.ReadPackets(packetChannel)
	close(packetChannel)
	wg.Wait()
	log.Println("Finished reading all packets.")

	// ---------------------------
	// Per-flow error calculation
	// ---------------------------
	file, err := os.Create("flow.txt")
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	countRelErrSum := 0.0
	sizeRelErrSum := 0.0

	for key, actualCount := range countMap {
		actualSize := sizeMap[key]
		flow := net.ParseIP(key).To16()

		result := task.Query(flow)
		estimatedCount := int(result >> 32)       // upper 32 bits
		estimatedSize := int(result & 0xffffffff) // lower 32 bits

		// Relative Error (Count)
		countRE := float64(estimatedCount-actualCount) / float64(actualCount)
		if countRE < 0 {
			countRE = -countRE
		}
		countRelErrSum += countRE

		// Relative Error (Size)
		if actualSize > 0 {
			sizeRE := float64(estimatedSize-actualSize) / float64(actualSize)
			if sizeRE < 0 {
				sizeRE = -sizeRE
			}
			sizeRelErrSum += sizeRE
		}

		_, err := fmt.Fprintf(writer,
			"%s %d %d %d %d\n",
			key, actualCount, estimatedCount, actualSize, estimatedSize)
		if err != nil {
			log.Fatalf("Failed to write: %v", err)
		}
	}

	avgCountRE := countRelErrSum / float64(len(countMap))
	avgSizeRE := sizeRelErrSum / float64(len(sizeMap))
	log.Printf("Per-flow Avg Relative Error (Count): %.4f", avgCountRE)
	log.Printf("Per-flow Avg Relative Error (Size): %.4f", avgSizeRE)

	// ---------------------------
	// Heavy Hitters (Count + Size)
	// ---------------------------
	hhs := task.Snapshot()
	if reflect.TypeOf(hhs) != reflect.TypeOf(statistic.HeavyRecord{}) {
		log.Fatalf("Unexpected type: %v", reflect.TypeOf(hhs))
	}
	if hhs == nil {
		log.Println("No heavy hitters detected.")
		return
	}
	res := hhs.(statistic.HeavyRecord)

	// Ground truth Count heavy hitters
	trueCountHH := make(map[string]int)
	for key, count := range countMap {
		if uint32(count) >= CountThreshold {
			trueCountHH[key] = count
		}
	}
	// Ground truth Size heavy hitters
	trueSizeHH := make(map[string]int)
	for key, size := range sizeMap {
		if uint32(size) >= SizeThreshold {
			trueSizeHH[key] = size
		}
	}

	// Detected Count heavy hitters
	detectedCountHH := make(map[string]uint32)
	for _, record := range res.Count {
		key := net.IP(record.Flow).String()
		detectedCountHH[key] = record.Count
	}
	// Detected Size heavy hitters
	detectedSizeHH := make(map[string]uint32)
	for _, record := range res.Size {
		key := net.IP(record.Flow).String()
		detectedSizeHH[key] = record.Size
	}

	// Evaluate Count HH
	countMRE, countPrec, countRec, countF1, tpC, fpC, fnC :=
		evaluateHeavyHitters(detectedCountHH, trueCountHH)
	log.Printf("[Count-HH] TP=%d FP=%d FN=%d", tpC, fpC, fnC)
	log.Printf("[Count-HH] MRE=%.4f Precision=%.4f Recall=%.4f F1=%.4f",
		countMRE, countPrec, countRec, countF1)

	// Evaluate Size HH
	sizeMRE, sizePrec, sizeRec, sizeF1, tpS, fpS, fnS :=
		evaluateHeavyHitters(detectedSizeHH, trueSizeHH)
	log.Printf("[Size-HH] TP=%d FP=%d FN=%d", tpS, fpS, fnS)
	log.Printf("[Size-HH] MRE=%.4f Precision=%.4f Recall=%.4f F1=%.4f",
		sizeMRE, sizePrec, sizeRec, sizeF1)
}
