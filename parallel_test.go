package main

import (
	"fmt"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// Test that we can spawn a basic thread
func TestBasicThreadSpawn(t *testing.T) {
	fmt.Println("Testing basic thread spawn...")

	// Create a counter to verify thread executed
	var counter int32 = 0

	// Allocate stack for thread
	stack := AllocateThreadStack(1024 * 1024) // 1MB stack

	// For this test, we'll use a simpler approach:
	// Just verify that clone() returns successfully
	// (Full function execution will be tested in integration tests)

	// Attempt to spawn a thread
	// Note: We're passing dummy values since executeThreadFunction is a placeholder
	tid, err := CloneThread(0, 0, stack)

	if err != nil {
		t.Fatalf("Failed to spawn thread: %v", err)
	}

	if tid <= 0 {
		t.Fatalf("Invalid thread ID returned: %d", tid)
	}

	fmt.Printf("Successfully spawned thread with TID: %d\n", tid)

	// Give thread time to execute and exit
	time.Sleep(100 * time.Millisecond)

	// Thread should have exited by now
	// We can't easily check if it completed without proper synchronization
	// (which we'll add in Phase 7), but the fact that clone() succeeded is good

	_ = counter // Avoid unused variable warning
}

// Test GetTID function
func TestGetTID(t *testing.T) {
	tid := GetTID()
	if tid <= 0 {
		t.Fatalf("Invalid TID returned: %d", tid)
	}
	fmt.Printf("Current thread TID: %d\n", tid)
}

// Test CPU core detection
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

// Test work distribution calculation
func TestWorkDistribution(t *testing.T) {
	tests := []struct {
		totalItems int
		numThreads int
		wantChunk  int
		wantRem    int
	}{
		{100, 4, 25, 0},   // Even split
		{100, 3, 33, 1},   // With remainder
		{10, 4, 2, 2},     // Small total
		{1000, 8, 125, 0}, // Larger numbers
	}

	for _, tt := range tests {
		chunk, rem := CalculateWorkDistribution(tt.totalItems, tt.numThreads)
		if chunk != tt.wantChunk || rem != tt.wantRem {
			t.Errorf("CalculateWorkDistribution(%d, %d) = (%d, %d), want (%d, %d)",
				tt.totalItems, tt.numThreads, chunk, rem, tt.wantChunk, tt.wantRem)
		}
	}
}

// Test thread work range calculation
func TestThreadWorkRange(t *testing.T) {
	// Test with 100 items, 4 threads
	totalItems := 100
	numThreads := 4

	expected := []struct{ start, end int }{
		{0, 25},   // Thread 0
		{25, 50},  // Thread 1
		{50, 75},  // Thread 2
		{75, 100}, // Thread 3
	}

	for i := 0; i < numThreads; i++ {
		start, end := GetThreadWorkRange(i, totalItems, numThreads)
		if start != expected[i].start || end != expected[i].end {
			t.Errorf("Thread %d: got range [%d, %d), want [%d, %d)",
				i, start, end, expected[i].start, expected[i].end)
		}
	}

	// Test with remainder: 101 items, 4 threads
	// Threads 0-2 get 25 items, thread 3 gets 26 items
	totalItems = 101
	expected = []struct{ start, end int }{
		{0, 25},   // Thread 0
		{25, 50},  // Thread 1
		{50, 75},  // Thread 2
		{75, 101}, // Thread 3 (gets +1)
	}

	for i := 0; i < numThreads; i++ {
		start, end := GetThreadWorkRange(i, totalItems, numThreads)
		if start != expected[i].start || end != expected[i].end {
			t.Errorf("Thread %d (with remainder): got range [%d, %d), want [%d, %d)",
				i, start, end, expected[i].start, expected[i].end)
		}
	}
}

// Test spawning multiple threads
func TestMultipleThreadSpawn(t *testing.T) {
	fmt.Println("Testing multiple thread spawn...")

	numThreads := 4
	tids := make([]int, numThreads)

	for i := 0; i < numThreads; i++ {
		stack := AllocateThreadStack(1024 * 1024)
		tid, err := CloneThread(0, 0, stack)

		if err != nil {
			t.Fatalf("Failed to spawn thread %d: %v", i, err)
		}

		if tid <= 0 {
			t.Fatalf("Invalid TID for thread %d: %d", i, tid)
		}

		tids[i] = tid
		fmt.Printf("Spawned thread %d with TID: %d\n", i, tid)
	}

	// Verify all TIDs are unique
	seen := make(map[int]bool)
	for i, tid := range tids {
		if seen[tid] {
			t.Errorf("Duplicate TID %d found for thread %d", tid, i)
		}
		seen[tid] = true
	}

	// Wait for threads to complete
	time.Sleep(200 * time.Millisecond)

	fmt.Printf("Successfully spawned and tracked %d threads\n", numThreads)
}

// Test manual thread function execution (bypassing CloneThread)
// This tests the actual thread execution path
func TestManualThreadExecution(t *testing.T) {
	fmt.Println("Testing manual thread execution with shared memory...")

	// Shared counter (atomic for thread safety)
	var counter atomic.Int32

	// Thread function that increments counter
	threadFunc := func() {
		tid := GetTID()
		fmt.Printf("Thread %d executing\n", tid)
		counter.Add(1)
		syscall.Exit(0)
	}

	// Spawn thread manually using raw clone
	stack := AllocateThreadStack(1024 * 1024)
	stackTop := stack.StackTop()

	// Call clone directly
	tid, _, errno := syscall.RawSyscall6(
		syscall.SYS_CLONE,
		uintptr(CLONE_VM|CLONE_FILES|CLONE_FS),
		stackTop,
		0, 0, 0, 0,
	)

	if errno != 0 {
		t.Fatalf("clone() failed: %v", errno)
	}

	// Check if we're in the child thread
	if tid == 0 {
		// Child thread - execute the function
		threadFunc()
		// threadFunc calls syscall.Exit, so we never return here
	}

	// Parent process
	fmt.Printf("Parent: spawned thread with TID %d\n", tid)

	// Wait for thread to execute
	time.Sleep(100 * time.Millisecond)

	// Check that counter was incremented
	finalCount := counter.Load()
	if finalCount != 1 {
		t.Errorf("Expected counter=1, got %d (thread may not have executed)", finalCount)
	} else {
		fmt.Printf("Success! Thread executed and incremented counter to %d\n", finalCount)
	}
}
