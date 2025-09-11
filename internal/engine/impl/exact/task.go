package exact

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/exact/statistic"
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
	"fmt"
	"hash/fnv"
	"log"
	"strconv"
	"strings"
	"time"
)

// --- Factory Registration ---

func init() {
	factory.RegisterAggregator("exact", func(cfg *config.Config) ([]model.Task, []model.Writer, error) {
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

		return tasks, writers, nil
	})
}

// --- Task Implementation ---

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

// Snapshot returns a deep copy of the current aggregated data without modifying the internal state, and is safe for concurrent use.
func (t *Task) Snapshot() interface{} {
	snapshotShards := make([]*statistic.Shard, t.shardCount)
	for i := 0; i < int(t.shardCount); i++ {
		shard := t.shards[i]
		shard.Mu.RLock() // Use RLock for read-only access

		// Deep copy the flows map
		copiedFlows := make(map[string]*statistic.Flow, len(shard.Flows))
		for k, v := range shard.Flows {
			// Copy the Flow struct itself to ensure independence
			flowCopy := *v
			copiedFlows[k] = &flowCopy
		}
		shard.Mu.RUnlock()

		snapshotShards[i] = &statistic.Shard{
			Flows: copiedFlows,
		}
	}
	return statistic.SnapshotData{
		TaskName: t.name,
		Shards:   snapshotShards,
	}
}

// Reset clears the internal state of the task, preparing for a new measurement period.
func (t *Task) Reset() {
	for i := 0; i < int(t.shardCount); i++ {
		shard := t.shards[i]
		shard.Mu.Lock()
		shard.Flows = make(map[string]*statistic.Flow) // Reset with a new empty map
		shard.Mu.Unlock()
	}
	log.Printf("Task '%s' state has been reset.", t.name)
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
