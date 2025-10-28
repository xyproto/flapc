package main

import (
	"fmt"
	"sync/atomic"
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

// GetNumCPUCores returns the number of online CPU cores
// Uses sysconf(_SC_NPROCESSORS_ONLN) via get_nprocs()
func GetNumCPUCores() int {
	// Read /proc/cpuinfo to count processors
	// This is more portable than sysconf and works in all Linux environments
	data, err := syscall.Open("/proc/cpuinfo", syscall.O_RDONLY, 0)
	if err != nil {
		// Fallback: assume 4 cores if we can't detect
		return 4
	}
	defer syscall.Close(data)

	// Read file contents
	buf := make([]byte, 16384) // 16KB should be enough for cpuinfo
	n, err := syscall.Read(data, buf)
	if err != nil {
		return 4
	}

	// Count "processor" lines in /proc/cpuinfo
	count := 0
	for i := 0; i < n-9; i++ {
		// Look for "processor" at start of line
		if buf[i] == 'p' && buf[i+1] == 'r' && buf[i+2] == 'o' &&
			buf[i+3] == 'c' && buf[i+4] == 'e' && buf[i+5] == 's' &&
			buf[i+6] == 's' && buf[i+7] == 'o' && buf[i+8] == 'r' {
			// Check if this is at start of line (i==0 or previous char is newline)
			if i == 0 || buf[i-1] == '\n' {
				count++
			}
		}
	}

	if count == 0 {
		return 4 // Fallback
	}
	return count
}

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

// Futex operations
const (
	FUTEX_WAIT            = 0
	FUTEX_WAKE            = 1
	FUTEX_PRIVATE_FLAG    = 128
	FUTEX_WAIT_PRIVATE    = FUTEX_WAIT | FUTEX_PRIVATE_FLAG
	FUTEX_WAKE_PRIVATE    = FUTEX_WAKE | FUTEX_PRIVATE_FLAG
)

// FutexWait atomically checks if *addr == val, and if so, sleeps until woken
// Returns 0 on success, error on failure
func FutexWait(addr *int32, val int32) error {
	_, _, errno := syscall.Syscall6(
		syscall.SYS_FUTEX,
		uintptr(unsafe.Pointer(addr)),
		uintptr(FUTEX_WAIT_PRIVATE),
		uintptr(val),
		0, // timeout (NULL = infinite)
		0, 0,
	)
	if errno != 0 && errno != syscall.EAGAIN {
		return errno
	}
	return nil
}

// FutexWake wakes up to count threads waiting on *addr
// Returns number of threads woken (or error)
func FutexWake(addr *int32, count int) (int, error) {
	n, _, errno := syscall.Syscall6(
		syscall.SYS_FUTEX,
		uintptr(unsafe.Pointer(addr)),
		uintptr(FUTEX_WAKE_PRIVATE),
		uintptr(count),
		0, 0, 0,
	)
	if errno != 0 {
		return 0, errno
	}
	return int(n), nil
}

// AtomicDecrement atomically decrements a counter and returns the new value
func AtomicDecrement(addr *int32) int32 {
	// Use sync/atomic which compiles to LOCK XADD on x86-64
	return atomic.AddInt32(addr, -1)
}

// Barrier represents a thread barrier for N threads
type Barrier struct {
	count   int32  // Number of threads that still need to arrive
	total   int32  // Total number of threads
}

// NewBarrier creates a barrier for numThreads threads
func NewBarrier(numThreads int) *Barrier {
	return &Barrier{
		count: int32(numThreads),
		total: int32(numThreads),
	}
}

// Wait blocks until all threads reach the barrier
func (b *Barrier) Wait() {
	// Atomically decrement counter
	remaining := AtomicDecrement(&b.count)

	if remaining == 0 {
		// Last thread: wake everyone
		FutexWake(&b.count, int(b.total))
		return
	}

	// Not last thread: wait until woken
	for atomic.LoadInt32(&b.count) > 0 {
		FutexWait(&b.count, remaining)
		// Check again in case of spurious wakeup
		remaining = atomic.LoadInt32(&b.count)
	}
}

// WaitForThreads waits for all threads in threadIDs to complete
// Uses a barrier-based approach
func WaitForThreads(barrier *Barrier) error {
	if barrier == nil {
		return fmt.Errorf("barrier is nil")
	}

	barrier.Wait()
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
