//go:build !linux
// +build !linux

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBasicThreadSpawn(t *testing.T) {
	t.Skip("Thread spawning not supported on this platform")
}

func TestGetTID(t *testing.T) {
	t.Skip("GetTID not supported on this platform")
}

func TestGetNumCPUCores(t *testing.T) {
	cores := GetNumCPUCores()
	if cores <= 0 {
		t.Fatalf("Invalid core count returned: %d", cores)
	}
	if cores > 1024 {
		t.Fatalf("Unrealistic core count returned: %d", cores)
	}
	fmt.Printf("Detected CPU cores: %d\n", cores)
}

func TestWorkDistribution(t *testing.T) {
	tests := []struct {
		totalItems int
		numThreads int
		wantChunk  int
		wantRem    int
	}{
		{100, 4, 25, 0},
		{100, 3, 33, 1},
		{10, 4, 2, 2},
		{1000, 8, 125, 0},
	}

	for _, tt := range tests {
		chunk, rem := CalculateWorkDistribution(tt.totalItems, tt.numThreads)
		if chunk != tt.wantChunk || rem != tt.wantRem {
			t.Errorf("CalculateWorkDistribution(%d, %d) = (%d, %d), want (%d, %d)",
				tt.totalItems, tt.numThreads, chunk, rem, tt.wantChunk, tt.wantRem)
		}
	}
}

func TestThreadWorkRange(t *testing.T) {
	totalItems := 100
	numThreads := 4

	expected := []struct{ start, end int }{
		{0, 25},
		{25, 50},
		{50, 75},
		{75, 100},
	}

	for i := 0; i < numThreads; i++ {
		start, end := GetThreadWorkRange(i, totalItems, numThreads)
		if start != expected[i].start || end != expected[i].end {
			t.Errorf("Thread %d: got range [%d, %d), want [%d, %d)",
				i, start, end, expected[i].start, expected[i].end)
		}
	}

	totalItems = 101
	expected = []struct{ start, end int }{
		{0, 25},
		{25, 50},
		{50, 75},
		{75, 101},
	}

	for i := 0; i < numThreads; i++ {
		start, end := GetThreadWorkRange(i, totalItems, numThreads)
		if start != expected[i].start || end != expected[i].end {
			t.Errorf("Thread %d (with remainder): got range [%d, %d), want [%d, %d)",
				i, start, end, expected[i].start, expected[i].end)
		}
	}
}

func TestMultipleThreadSpawn(t *testing.T) {
	t.Skip("Thread spawning not supported on this platform")
}

func TestBarrierSync(t *testing.T) {
	fmt.Println("Testing barrier synchronization...")

	numThreads := 4
	barrier := NewBarrier(numThreads)
	counter := atomic.Int32{}

	completionOrder := make([]int, 0, numThreads)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			counter.Add(1)
			time.Sleep(time.Duration(id*10) * time.Millisecond)

			fmt.Printf("Thread %d reached barrier\n", id)
			barrier.Wait()

			mu.Lock()
			completionOrder = append(completionOrder, id)
			mu.Unlock()

			fmt.Printf("Thread %d passed barrier\n", id)
		}(i)
	}

	wg.Wait()

	if len(completionOrder) != numThreads {
		t.Errorf("Expected %d threads to complete, got %d", numThreads, len(completionOrder))
	}

	finalCount := counter.Load()
	if finalCount != int32(numThreads) {
		t.Errorf("Expected counter=%d, got %d", numThreads, finalCount)
	}

	fmt.Printf("Success! All %d threads synchronized at barrier\n", numThreads)
}

func TestManualThreadExecution(t *testing.T) {
	t.Skip("Manual thread execution not supported on this platform")
}
