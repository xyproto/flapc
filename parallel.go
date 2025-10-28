package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Linux clone() flags for thread creation
const (
	CLONE_VM             = 0x00000100 // Share memory space (threads)
	CLONE_FS             = 0x00000200 // Share filesystem information
	CLONE_FILES          = 0x00000400 // Share file descriptor table
	CLONE_SIGHAND        = 0x00000800 // Share signal handlers
	CLONE_THREAD         = 0x00010000 // Create a thread (same thread group)
	CLONE_SYSVSEM        = 0x00040000 // Share System V SEM_UNDO semantics
	CLONE_SETTLS         = 0x00080000 // Set thread local storage
	CLONE_PARENT_SETTID  = 0x00100000 // Write child TID to parent's memory
	CLONE_CHILD_CLEARTID = 0x00200000 // Clear child TID and wake futex
)

// Standard thread creation flags (same as pthread_create uses)
const CLONE_THREAD_FLAGS = CLONE_VM | CLONE_FS | CLONE_FILES | CLONE_SIGHAND |
                           CLONE_THREAD | CLONE_SYSVSEM

// ThreadStack represents a thread's stack allocation
type ThreadStack struct {
	Memory []byte
	Size   int
}

// AllocateThreadStack allocates a new stack for a thread
// Default Linux thread stack size is typically 8MB
func AllocateThreadStack(size int) *ThreadStack {
	if size <= 0 {
		size = 8 * 1024 * 1024 // 8MB default
	}

	return &ThreadStack{
		Memory: make([]byte, size),
		Size:   size,
	}
}

// StackTop returns the top of the stack (stacks grow downward)
func (ts *ThreadStack) StackTop() uintptr {
	return uintptr(unsafe.Pointer(&ts.Memory[ts.Size-16])) // -16 for alignment
}

// CloneThread spawns a new thread using the clone() syscall
// Returns the thread ID (TID) of the new thread, or error
//
// fn: function pointer to execute in the new thread
// arg: argument to pass to the function
// stack: pre-allocated stack for the thread
func CloneThread(fn uintptr, arg uintptr, stack *ThreadStack) (int, error) {
	// Get stack top (stacks grow downward on x86-64)
	stackTop := stack.StackTop()

	// Call clone() syscall
	// syscall: clone(flags, stack, ptid, ctid, tls)
	tid, _, errno := syscall.RawSyscall6(
		syscall.SYS_CLONE,
		uintptr(CLONE_THREAD_FLAGS),
		stackTop,
		0, // ptid (not used without CLONE_PARENT_SETTID)
		0, // ctid (not used without CLONE_CHILD_CLEARTID)
		0, // tls (not setting thread-local storage)
		0,
	)

	if errno != 0 {
		return -1, fmt.Errorf("clone() failed: %v", errno)
	}

	// In the parent process, tid > 0
	// In the child thread, tid == 0
	if tid == 0 {
		// We're in the child thread - execute the function
		// Call the function pointer with the argument
		executeThreadFunction(fn, arg)

		// Thread function returned - exit this thread
		syscall.Exit(0)
	}

	// Parent process: return child TID
	return int(tid), nil
}

// executeThreadFunction calls a function pointer with an argument
// This is a bridge function to handle the calling convention
func executeThreadFunction(fn uintptr, arg uintptr) {
	// TODO: In the actual code generation phase, we'll jump directly to
	// the compiled loop body function. For now, this is a placeholder.

	// For testing, we can use a simple approach:
	// Convert fn to a Go function and call it
	// (This will be replaced with direct assembly jumps in code generation)

	fmt.Printf("Thread started (placeholder - fn: %x, arg: %x)\n", fn, arg)
}

// GetTID returns the current thread's ID using the gettid() syscall
func GetTID() int {
	tid, _, _ := syscall.RawSyscall(syscall.SYS_GETTID, 0, 0, 0)
	return int(tid)
}

// WaitForThreads implements a simple barrier using futex
// This will wait until all threads complete
func WaitForThreads(threadIDs []int) error {
	// TODO: Implement futex-based barrier for Phase 7
	// For now, use a simple sleep to allow threads to execute
	// (This is a placeholder - proper synchronization will be added later)

	fmt.Printf("Waiting for %d threads to complete...\n", len(threadIDs))

	// Placeholder: In production, this would be a futex wait
	// For now, we'll implement proper synchronization in Phase 7

	return nil
}

// CalculateWorkDistribution calculates how to split work across threads
// Returns: chunkSize, remainder
func CalculateWorkDistribution(totalItems int, numThreads int) (int, int) {
	if numThreads <= 0 {
		numThreads = 1
	}

	chunkSize := totalItems / numThreads
	remainder := totalItems % numThreads

	return chunkSize, remainder
}

// GetThreadWorkRange calculates the start and end indices for a thread's work
// threadID: 0-based thread index (0, 1, 2, ...)
// totalItems: total number of items to process
// numThreads: total number of threads
func GetThreadWorkRange(threadID int, totalItems int, numThreads int) (int, int) {
	chunkSize, remainder := CalculateWorkDistribution(totalItems, numThreads)

	startIdx := threadID * chunkSize
	endIdx := startIdx + chunkSize

	// Last thread gets the remainder items
	if threadID == numThreads-1 {
		endIdx += remainder
	}

	return startIdx, endIdx
}
