package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"time"
)

type Flow struct {
	Key         string
	StartTime   time.Time
	EndTime     time.Time
	ByteCount   uint64
	PacketCount uint64
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./scripts/gobana/main.go <gob_file>")
		os.Exit(1)
	}
	gobFile := os.Args[1]

    file, err := os.Open(gobFile)
    if err != nil {
        log.Fatalf("Unable to open file: %v", err)
    }
    defer file.Close()

    decoder := gob.NewDecoder(file)

    var mp map[string]Flow
    
    err = decoder.Decode(&mp)
    if err != nil {
        log.Fatalf("Failed to decode gob data: %v", err)
    }

    fmt.Println("Decoded Flows:")
    fmt.Printf("%+v\n", mp)
}