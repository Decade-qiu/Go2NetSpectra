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
	"slices"
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

	packetChannel := make(chan *v1.PacketInfo, 10000)

	Counthreshold := uint32(4096)
	Sizethreshold := uint32(4096 * 1024)

	// Initialize CountMin sketch
	task := New("per_src_flow",
		[]string{"SrcIP"},
		[]string{"DstIP", "SrcPort", "DstPort", "Protocol"},
		1<<13, 2, Sizethreshold, Counthreshold)

	// Ground truth (map-based)
	countMap := make(map[string]int)
	sizeMap := make(map[string]int)

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
			countMap[key]++
			sizeMap[key] += info.Length
		}
	})

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

		_, err := writer.WriteString(
			fmt.Sprintf("%s count=%d est=%d size=%d est=%d\n",
				key, actualCount, estimatedCount, actualSize, estimatedSize))
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
	res := task.Snapshot().(statistic.HeavyRecord)

	// Ground truth Count heavy hitters
	trueCountHH := make(map[string]int)
	for key, count := range countMap {
		if uint32(count) >= Counthreshold {
			trueCountHH[key] = count
		}
	}
	// Ground truth Size heavy hitters
	trueSizeHH := make(map[string]int)
	for key, size := range sizeMap {
		if uint32(size) >= Sizethreshold {
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

func evaluateHeavyHitters(detected map[string]uint32, truth map[string]int) (mre, precision, recall, f1 float64, tp, fp, fn int) {
	mreSum := 0.0

	// Compare detected with ground truth
	for key, estVal := range detected {
		if actualVal, isTrue := truth[key]; isTrue {
			// True Positive
			tp++
			relativeError := float64(int(estVal)-actualVal) / float64(actualVal)
			if relativeError < 0 {
				relativeError = -relativeError
			}
			mreSum += relativeError
		} else {
			// False Positive
			fp++
		}
	}

	// False Negatives = true HHs not detected
	for key := range truth {
		if _, found := detected[key]; !found {
			fn++
		}
	}

	// Calculate metrics
	if tp > 0 {
		mre = mreSum / float64(tp)
	}
	if (tp + fp) > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	if (tp + fn) > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	if (precision + recall) > 0 {
		f1 = 2 * (precision * recall) / (precision + recall)
	}

	printTopN := func(m map[string]int, n int, title string) {
		type kv struct {
			Key   string
			Value int
		}
		var arr []kv
		for k, v := range m {
			arr = append(arr, kv{k, v})
		}
		slices.SortFunc(arr, func(a, b kv) int { return b.Value - a.Value }) // descending
		if len(arr) > n {
			arr = arr[:n]
		}
		fmt.Printf("%s:\n", title)
		for _, kv := range arr {
			fmt.Printf("  %s -> %d\n", kv.Key, kv.Value)
		}
	}

	// Convert detected from uint32 to int for printing
	detectedInt := make(map[string]int, len(detected))
	for k, v := range detected {
		detectedInt[k] = int(v)
	}

	printTopN(truth, 5, "Top-5 Ground Truth HHs")
	printTopN(detectedInt, 5, "Top-5 Detected HHs")

	return
}
