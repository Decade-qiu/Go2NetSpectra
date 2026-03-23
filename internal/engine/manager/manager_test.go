package manager

import (
	"net"
	"sync"
	"testing"
	"time"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
)

type stubTask struct {
	mu      sync.Mutex
	packets []*model.PacketInfo
}

func (s *stubTask) ProcessPacket(packet *model.PacketInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyPacket := *packet
	s.packets = append(s.packets, &copyPacket)
}

func (s *stubTask) Snapshot() interface{} { return nil }

func (s *stubTask) Reset() {}

func (s *stubTask) Name() string { return "stub" }

func (s *stubTask) Query(flow []byte) uint64 { return 0 }

func (s *stubTask) Fields() []string { return nil }

func (s *stubTask) DecodeFlowFunc() func(flow []byte, fields []string) string {
	return func(flow []byte, fields []string) string { return "" }
}

func (s *stubTask) AlerterMsg(rules []config.AlerterRule) string { return "" }

func (s *stubTask) packetCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.packets)
}

func (s *stubTask) firstPacket() *model.PacketInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.packets) == 0 {
		return nil
	}
	return s.packets[0]
}

func TestManagerProcessPacketRoutesToAllTasks(t *testing.T) {
	taskA := &stubTask{}
	taskB := &stubTask{}
	m := &Manager{
		taskGroups: []factory.TaskGroup{
			{Tasks: []model.Task{taskA, taskB}},
		},
	}

	packet := &model.PacketInfo{
		Timestamp: time.Unix(1700000000, 0),
		FiveTuple: model.FiveTuple{
			SrcIP:    net.IP{192, 0, 2, 10},
			DstIP:    net.IP{198, 51, 100, 20},
			SrcPort:  443,
			DstPort:  8443,
			Protocol: 6,
		},
		Length: 256,
	}

	if err := m.processPacket(packet); err != nil {
		t.Fatalf("processPacket() unexpected error: %v", err)
	}

	for name, task := range map[string]*stubTask{"taskA": taskA, "taskB": taskB} {
		if got := task.packetCount(); got != 1 {
			t.Fatalf("%s packet count = %d, want 1", name, got)
		}
		gotPacket := task.firstPacket()
		if gotPacket == nil {
			t.Fatalf("%s first packet = nil, want non-nil", name)
		}
		if gotPacket.Length != 256 {
			t.Fatalf("%s first packet length = %d, want 256", name, gotPacket.Length)
		}
	}
}

func TestManagerStopDrainsQueuedPackets(t *testing.T) {
	task := &stubTask{}
	m := &Manager{
		taskGroups: []factory.TaskGroup{
			{Tasks: []model.Task{task}},
		},
		packetChannel: make(chan *model.PacketInfo, 1),
		done:          make(chan struct{}),
		numWorkers:    1,
	}

	m.workerWg.Add(1)
	go m.worker()

	m.packetChannel <- &model.PacketInfo{
		Timestamp: time.Unix(1700000001, 0),
		FiveTuple: model.FiveTuple{
			SrcIP:    net.IP{10, 0, 0, 1},
			DstIP:    net.IP{10, 0, 0, 2},
			SrcPort:  1234,
			DstPort:  80,
			Protocol: 6,
		},
		Length: 64,
	}

	m.Stop()

	if got := task.packetCount(); got != 1 {
		t.Fatalf("processed packet count = %d, want 1", got)
	}
}
