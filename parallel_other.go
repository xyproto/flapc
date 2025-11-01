//go:build !linux
// +build !linux

package main

import (
	"fmt"
	"runtime"
	"sync/atomic"
)

func GetNumCPUCores() int {
	return runtime.NumCPU()
}

type ThreadStack struct {
	Memory []byte
	Size   int
}

func AllocateThreadStack(size int) *ThreadStack {
	return &ThreadStack{
		Memory: make([]byte, size),
		Size:   size,
	}
}

func (ts *ThreadStack) StackTop() uintptr {
	return 0
}

func CloneThread(fn uintptr, arg uintptr, stack *ThreadStack) (int, error) {
	return -1, fmt.Errorf("thread cloning not supported on this platform")
}

func executeThreadFunction(fn uintptr, arg uintptr) {
}

func GetTID() int {
	return 0
}

func FutexWait(addr *int32, val int32) error {
	return fmt.Errorf("futex not supported on this platform")
}

func FutexWake(addr *int32, count int) (int, error) {
	return 0, fmt.Errorf("futex not supported on this platform")
}

func AtomicDecrement(addr *int32) int32 {
	return atomic.AddInt32(addr, -1)
}

type Barrier struct {
	count int32
	total int32
}

func NewBarrier(numThreads int) *Barrier {
	return &Barrier{
		count: int32(numThreads),
		total: int32(numThreads),
	}
}

func (b *Barrier) Wait() {
	remaining := AtomicDecrement(&b.count)

	if remaining == 0 {
		return
	}

	for atomic.LoadInt32(&b.count) > 0 {
	}
}

func WaitForThreads(barrier *Barrier) error {
	if barrier == nil {
		return fmt.Errorf("barrier is nil")
	}

	barrier.Wait()
	return nil
}

func CalculateWorkDistribution(totalItems int, numThreads int) (int, int) {
	if numThreads <= 0 {
		numThreads = 1
	}

	chunkSize := totalItems / numThreads
	remainder := totalItems % numThreads

	return chunkSize, remainder
}

func GetThreadWorkRange(threadID int, totalItems int, numThreads int) (int, int) {
	chunkSize, remainder := CalculateWorkDistribution(totalItems, numThreads)

	startIdx := threadID * chunkSize
	endIdx := startIdx + chunkSize

	if threadID == numThreads-1 {
		endIdx += remainder
	}

	return startIdx, endIdx
}
