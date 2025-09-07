package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// KeyedAggregator performs aggregation for a specific set of key fields.
// It is responsible for a single task defined in the config.
type KeyedAggregator struct {
	Name        string
	KeyFields   []string
	activeFlows map[string]*model.Flow
	mu          sync.RWMutex
}

// NewKeyedAggregator creates a new aggregator for a specific keying configuration.
func NewKeyedAggregator(name string, keyFields []string) *KeyedAggregator {
	return &KeyedAggregator{
		Name:        name,
		KeyFields:   keyFields,
		activeFlows: make(map[string]*model.Flow),
	}
}

// generateKey creates a unique string key for a packet based on the aggregator's KeyFields.
func (ka *KeyedAggregator) generateKey(ft model.FiveTuple) (string, error) {
	var parts []string
	for _, field := range ka.KeyFields {
		switch field {
		case "SrcIP":
			parts = append(parts, ft.SrcIP.String())
		case "DstIP":
			parts = append(parts, ft.DstIP.String())
		case "SrcPort":
			parts = append(parts, strconv.Itoa(int(ft.SrcPort)))
		case "DstPort":
			parts = append(parts, strconv.Itoa(int(ft.DstPort)))
		case "Protocol":
			parts = append(parts, strconv.Itoa(int(ft.Protocol)))
		default:
			return "", fmt.Errorf("unknown key field: %s", field)
		}
	}
	return strings.Join(parts, "-"), nil
}

// ProcessPacket processes a single packet, creating or updating a flow.
func (ka *KeyedAggregator) ProcessPacket(packetInfo *model.PacketInfo) {
	key, err := ka.generateKey(packetInfo.FiveTuple)
	if err != nil {
		// In a real application, we'd use a structured logger.
		fmt.Printf("Error generating key for aggregator '%s': %v\n", ka.Name, err)
		return
	}

	ka.mu.Lock()
	defer ka.mu.Unlock()

	if flow, ok := ka.activeFlows[key]; ok {
		// Flow exists, update it
		flow.EndTime = packetInfo.Timestamp
		flow.PacketCount++
		flow.ByteCount += uint64(packetInfo.Length)
	} else {
		// Flow does not exist, create a new one
		ka.activeFlows[key] = &model.Flow{
			Key:         key,
			StartTime:   packetInfo.Timestamp,
			EndTime:     packetInfo.Timestamp,
			PacketCount: 1,
			ByteCount:   uint64(packetInfo.Length),
		}
	}
}
