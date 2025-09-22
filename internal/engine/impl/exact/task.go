package exact

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/exact/statistic"
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Factory Registration ---

func init() {
	factory.RegisterAggregator("exact", func(cfg *config.Config) (*factory.TaskGroup, error) {
		exactCfg := cfg.Aggregator.Exact

		// Create all enabled writers for this aggregator group
		writers := make([]model.Writer, 0, len(exactCfg.Writers))
		for _, writerDef := range exactCfg.Writers {
			if !writerDef.Enabled {
				continue
			}

			interval, err := time.ParseDuration(writerDef.SnapshotInterval)
			if err != nil {
				log.Printf("Warning: invalid snapshot_interval for writer type '%s': %v, skipping.", writerDef.Type, err)
				continue
			}

			var writer model.Writer
			switch writerDef.Type {
			case "gob":
				writer = NewGobWriter(writerDef.Gob.RootPath, interval)
			case "clickhouse":
				writer, err = NewClickHouseWriter(writerDef.ClickHouse, interval)
				if err != nil {
					log.Printf("Warning: failed to create writer type '%s': %v, skipping.", writerDef.Type, err)
					continue
				}
			default:
				log.Printf("Warning: unknown writer type '%s' in config, skipping.", writerDef.Type)
				continue
			}
			writers = append(writers, writer)
		}

		// Create all tasks for this aggregator group
		tasks := make([]model.Task, len(exactCfg.Tasks))
		for i, taskCfg := range exactCfg.Tasks {
			tasks[i] = New(taskCfg.Name, taskCfg.KeyFields, taskCfg.NumShards)
		}

		return &factory.TaskGroup{Tasks: tasks, Writers: writers}, nil
	})
}

// --- Task Implementation ---

const (
	IPv4ByteSize = 4
	IPv6ByteSize = 16
	PortByteSize = 2
	ProtoByteSize = 1
)

const defaultShardCount = 256

// Task performs exact aggregation for a specific set of key fields using a sharded map.
// It implements the model.Task interface.
type Task struct {
	name        string
	keyFields   []string
	shards      []*statistic.Shard
	shardCount  uint32
}

// New creates a new exact aggregation task.
func New(name string, keyFields []string, numShards uint32) model.Task {
	if numShards <= 0 || numShards >= 32768 {
		numShards = defaultShardCount
	}
	log.Printf("Creating ExactTask '%s' with %d shards for keys: %v", name, numShards, keyFields)
	task := &Task{
		name:        name,
		keyFields:   keyFields,
		shards:      make([]*statistic.Shard, numShards),
		shardCount:  numShards,
	}
	for i := 0; i < int(numShards); i++ {
		task.shards[i] = &statistic.Shard{
			Flows: make(map[string]*statistic.Flow),
		}
	}
	return task
}

// Name returns the name of the task.
func (t *Task) Name() string {
	return t.name
}

// Fields 
func (t *Task) Fields() []string {
	return []string{}
}

// Func 
func (t *Task) DecodeFlowFunc() func(flow []byte, fields []string) string {
	return func(flow []byte, fields []string) string {
		return ""
	}
}

// ProcessPacket processes a single packet, creating or updating a flow in the correct shard.
func (t *Task) ProcessPacket(packetInfo *model.PacketInfo) {
	fields, key, err := t.generateKeyAndFields(packetInfo.FiveTuple)
	if err != nil {
		log.Printf("Error generating key for task '%s': %v\n", t.name, err)
		return
	}

	shard := t.getShard(key)
	shard.Mu.Lock()
	defer shard.Mu.Unlock()

	if flow, ok := shard.Flows[key]; ok {
		flow.EndTime = packetInfo.Timestamp
		flow.PacketCount++
		flow.ByteCount += uint64(packetInfo.Length)
	} else {
		shard.Flows[key] = &statistic.Flow{
			Key:         key,
			Fields:      fields,
			StartTime:   packetInfo.Timestamp,
			EndTime:     packetInfo.Timestamp,
			PacketCount: 1,
			ByteCount:   uint64(packetInfo.Length),
		}
	}
}

// Snapshot returns a deep copy of the current aggregated data.
// Suitable for write-heavy, read-light scenarios.
// Concurrent writes are safe; snapshot reflects a consistent state at the moment of call.
func (t *Task) Snapshot() interface{} {
    snapshotShards := make([]*statistic.Shard, t.shardCount)
    var wg sync.WaitGroup
    wg.Add(int(t.shardCount)) // Wait for all shards to finish copying

    for i := 0; i < int(t.shardCount); i++ {
        go func(i int) {
            defer wg.Done()

            shard := t.shards[i]

            // Acquire read lock to safely read shard.Flows
            // Allows concurrent reads but blocks writes
            shard.Mu.RLock()

            // Deep copy the flows map to ensure the snapshot is independent
            copiedFlows := make(map[string]*statistic.Flow, len(shard.Flows))
            for k, v := range shard.Flows {
                // Copy each Flow struct to ensure modifications to original Flow
                // do not affect the snapshot
                flowCopy := *v
                copiedFlows[k] = &flowCopy
            }

            shard.Mu.RUnlock() // Release read lock

            // Store the shard snapshot
            snapshotShards[i] = &statistic.Shard{
                Flows: copiedFlows,
            }
        }(i)
    }

    wg.Wait() // Wait until all shard snapshots are complete

    // Return the full snapshot
    return statistic.SnapshotData{
        TaskName: t.name,
        Shards:   snapshotShards,
    }
}

// Reset clears the internal state of the task, preparing for a new measurement period.
func (t *Task) Reset() {
	var wait sync.WaitGroup
	wait.Add(int(t.shardCount)) // Wait for all shards to be reset

	for i := 0; i < int(t.shardCount); i++ {
		go func (i int) {
			defer wait.Done()
			shard := t.shards[i]
			shard.Mu.Lock()
			shard.Flows = make(map[string]*statistic.Flow) // Reset with a new empty map
			shard.Mu.Unlock()
		}(i)
	}

	wait.Wait() // Wait until all shards are reset
}

// AlerterMsg evaluates rules against the task's aggregated data and returns a markdown string if triggered.
func (t *Task) AlerterMsg(rules []config.AlerterRule) string {
	// Perform a snapshot to get the latest data for evaluation.
	snapshotData, ok := t.Snapshot().(statistic.SnapshotData)
	if !ok {
		log.Printf("ERROR: AlerterMsg in exact task received unexpected snapshot type: %T", t.Snapshot())
		return ""
	}

	// Calculate total metrics from the snapshot.
	var totalPackets uint64
	var totalBytes uint64
	flowCount := 0
	for _, shard := range snapshotData.Shards {
		for _, flow := range shard.Flows {
			totalPackets += flow.PacketCount
			totalBytes += flow.ByteCount
			flowCount++
		}
	}

	var triggeredMessages []string

	for _, rule := range rules {
		if rule.TaskName != t.name {
			continue
		}

		var triggered bool
		var currentValue float64
		var unit string

		switch rule.Metric {
		case "total_packets":
			currentValue = float64(totalPackets)
			unit = "packets"
			if check(currentValue, rule.Threshold, rule.Operator) {
				triggered = true
			}
		case "total_bytes":
			currentValue = float64(totalBytes)
			unit = "bytes"
			if check(currentValue, rule.Threshold, rule.Operator) {
				triggered = true
			}
		case "total_flows":
			currentValue = float64(flowCount)
			unit = "flows"
			if check(currentValue, rule.Threshold, rule.Operator) {
				triggered = true
			}
		}

		if triggered {
			msg := fmt.Sprintf("<h3>Alert: %s</h3>"+
				"<ul>"+
				"<li><b>Task:</b> <code>%s</code></li>"+
				"<li><b>Metric:</b> <code>%s</code></li>"+
				"<li><b>Condition:</b> <code>%s %.2f</code></li>"+
				"<li><b>Observed Value:</b> <code>%.0f %s</code></li>"+
				"</ul>",
				rule.Name, rule.TaskName, rule.Metric, rule.Operator, rule.Threshold, currentValue, unit)
			triggeredMessages = append(triggeredMessages, msg)
		}
	}

	return strings.Join(triggeredMessages, "<br><hr><br>")
}

// check compares a value against a threshold based on an operator.
func check(value, threshold float64, operator string) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case "=":
		return value == threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	default:
		log.Printf("Warning: unknown operator '%s' in alerter rule", operator)
		return false
	}
}

// Empty
// Just to satisfy the interface
func (t *Task) Query(flow []byte) uint64 {
	parts := make([]string, len(t.keyFields))
	index := 0
	for i, fieldName := range t.keyFields {
		switch fieldName {
		case "SrcIP", "DstIP":
			val := net.IP(flow[index : index+IPv6ByteSize]).String()
			parts[i] = val
			index += IPv6ByteSize
		case "SrcPort", "DstPort":
			val := uint16(flow[index])<<8 | uint16(flow[index+1])
			parts[i] = strconv.Itoa(int(val))
			index += PortByteSize
		case "Protocol":
			val := uint8(flow[index])
			parts[i] = strconv.Itoa(int(val))
			index += ProtoByteSize
		default:
			return 0
		}
	}
	key := strings.Join(parts, "-")
	shard := t.getShard(key)
	shard.Mu.RLock()
	defer shard.Mu.RUnlock()
	if flow, ok := shard.Flows[key]; ok {
		return flow.PacketCount << 32 | flow.ByteCount
	}
	return 0
}

// getShard returns the appropriate shard for a given key.
func (t *Task) getShard(key string) *statistic.Shard {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return t.shards[hasher.Sum32()%t.shardCount]
}

// generateKeyAndFields creates a unique string key and a field map for a packet.
func (t *Task) generateKeyAndFields(ft model.FiveTuple) (map[string]interface{}, string, error) {
	parts := make([]string, len(t.keyFields))
	fields := make(map[string]interface{}, len(t.keyFields))

	for i, fieldName := range t.keyFields {
		switch fieldName {
		case "SrcIP":
			val := ft.SrcIP.String()
			parts[i] = val
			fields[fieldName] = val
		case "DstIP":
			val := ft.DstIP.String()
			parts[i] = val
			fields[fieldName] = val
		case "SrcPort":
			val := ft.SrcPort
			parts[i] = strconv.Itoa(int(val))
			fields[fieldName] = val
		case "DstPort":
			val := ft.DstPort
			parts[i] = strconv.Itoa(int(val))
			fields[fieldName] = val
		case "Protocol":
			val := ft.Protocol
			parts[i] = strconv.Itoa(int(val))
			fields[fieldName] = val
		default:
			return nil, "", fmt.Errorf("unknown key field: %s", fieldName)
		}
	}
	return fields, strings.Join(parts, "-"), nil
}
