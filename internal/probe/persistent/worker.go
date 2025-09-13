package persistent

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

// PacketContainer holds both the raw packet and the parsed info.
type PacketContainer struct {
	RawPacket  gopacket.Packet
	PacketInfo *model.PacketInfo
}

// Worker manages a pool of goroutines for persistently writing packets to disk.
type Worker struct {
	packetChan chan *PacketContainer
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewWorker creates and starts a new persistent worker pool.
func NewWorker(cfg config.PersistenceConfig) (*Worker, error) {
	// Ensure the directory exists
	if err := os.MkdirAll(cfg.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create persistence directory: %w", err)
	}

	bufferSize := cfg.ChannelBufferSize
	if bufferSize <= 0 {
		bufferSize = 10000 // Default value
	}

	w := &Worker{
		packetChan: make(chan *PacketContainer, bufferSize),
		stopChan:   make(chan struct{}),
	}

	w.Start(cfg)
	return w, nil
}

// Start launches the worker goroutines.
func (w *Worker) Start(cfg config.PersistenceConfig) {
	file, err := w.createOutputFile(cfg)
	if err != nil {
		log.Fatalf("PersistentWorker: Failed to create output file: %v", err)
	}

	var workerFunc func(file *os.File)
	switch cfg.Encoding {
	case "gob":
		workerFunc = w.runGobWorker
	case "text":
		workerFunc = w.runTextWorker
	case "pcap":
		// pcap writer needs the link type, which we don't have here.
		// This is a limitation. We will assume Ethernet for now.
		// A better solution would be to pass the link type from the capture source.
		pcapWriter := pcapgo.NewWriter(file)
		if err := pcapWriter.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
			log.Fatalf("PersistentWorker (pcap): Failed to write file header: %v", err)
		}
		workerFunc = w.runPcapWorker(pcapWriter)
	default:
		log.Printf("PersistentWorker: Unknown encoding '%s', workers will not start.", cfg.Encoding)
		file.Close()
		return
	}

	numWorkers := cfg.NumWorkers
	if numWorkers <= 0 {
		numWorkers = 1 // pcap writing is often better single-threaded to avoid out-of-order packets
	}

	w.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer w.wg.Done()
			workerFunc(file)
		}()
	}

	go func() {
		<-w.stopChan
		close(w.packetChan)
		w.wg.Wait()
		if err := file.Close(); err != nil {
			log.Printf("PersistentWorker: Error closing file: %v", err)
		}
		log.Println("Persistent worker stopped and file closed.")
	}()

	log.Printf("Persistent worker started with %d goroutines, encoding: %s, writing to: %s", numWorkers, cfg.Encoding, file.Name())
}

func (w *Worker) createOutputFile(cfg config.PersistenceConfig) (*os.File, error) {
	ext := ".log"
	switch cfg.Encoding {
	case "gob":
		ext = ".gob"
	case "pcap":
		ext = ".pcap"
	}
	fileName := fmt.Sprintf("%s%s", time.Now().Format("2006-01-02_15-04-05"), ext)
	filePath := filepath.Join(cfg.Path, fileName)
	return os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
}

func (w *Worker) runGobWorker(file *os.File) {
	encoder := gob.NewEncoder(file)
	for container := range w.packetChan {
		if err := encoder.Encode(container.PacketInfo); err != nil {
			log.Printf("PersistentWorker (gob): Error encoding packet: %v", err)
		}
	}
}

func (w *Worker) runTextWorker(file *os.File) {
	writer := bufio.NewWriter(file)
	for container := range w.packetChan {
		packet := container.PacketInfo
		line := fmt.Sprintf("%s - %s:%d -> %s:%d, Proto: %d, Len: %d\n",
			packet.Timestamp.Format("2006-01-02 15:04:05.000"),
			packet.FiveTuple.SrcIP,
			packet.FiveTuple.SrcPort,
			packet.FiveTuple.DstIP,
			packet.FiveTuple.DstPort,
			packet.FiveTuple.Protocol,
			packet.Length,
		)
		if _, err := writer.WriteString(line); err != nil {
			log.Printf("PersistentWorker (text): Error writing packet: %v", err)
		}
	}
	writer.Flush()
}

func (w *Worker) runPcapWorker(pcapWriter *pcapgo.Writer) func(*os.File) {
	return func(file *os.File) {
		for container := range w.packetChan {
			if err := pcapWriter.WritePacket(container.RawPacket.Metadata().CaptureInfo, container.RawPacket.Data()); err != nil {
				log.Printf("PersistentWorker (pcap): Error writing packet: %v", err)
			}
		}
	}
}

// Stop gracefully shuts down the worker pool.
func (w *Worker) Stop() {
	close(w.stopChan)
}

// Enqueue sends a packet container to the worker channel for processing.
func (w *Worker) Enqueue(container *PacketContainer) {
	select {
	case w.packetChan <- container:
	default:
		log.Println("PersistentWorker: Channel is full, dropping packet.")
	}
}
