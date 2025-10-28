# Flapc TODO - Version 1.6 Release

**Target**: x86-64 Linux game development
**Philosophy**: Minimal, elegant, implementable

---

## 🔥 Critical Path to 1.6

### 1. Parallel Loops (CPU Parallelism)
Essential for game performance: physics, AI, rendering

**Phase 1: Lexer & Parser** ✅ COMPLETE
- [x] Add `@@` token to lexer (all cores syntax)
- [x] Parse `N @` prefix (e.g., `4 @ item in data { }`)
- [x] Handle `@@` as special case (detect cores at compile time)
- [x] Parse full loop body for parallel loops
- [x] Error handling for receive loops (not parallelizable)

**Phase 2: AST Changes** ✅ COMPLETE
- [x] Add `NumThreads` field to `LoopStmt` in ast.go
- [x] Update `String()` method for parallel loops
- [x] Update `substituteParamsStmt()` to preserve NumThreads

**Phase 3: Basic Thread Creation** ✅ COMPLETE
- [x] Create new `parallel.go` file (265 lines)
- [x] Add `clone()` syscall wrapper
- [x] Implement thread spawn with CLONE_VM flag
- [x] Test: spawn single thread, verified TID 2314102

**Phase 4: Thread ID & Verification** ✅ COMPLETE
- [x] Add syscall to get thread ID (GetTID)
- [x] Test: spawn 4 threads, each gets unique TID
- [x] Verify all threads run independently

**Phase 5: Work Distribution Math** ✅ COMPLETE
- [x] Calculate: `chunk_size = total_items / num_threads`
- [x] Calculate: `start_idx = thread_id * chunk_size`
- [x] Calculate: `end_idx = start_idx + chunk_size`
- [x] Handle remainder items (give to last thread)
- [x] Test coverage: 100÷2=50, 100÷4=25, 101÷4 (with remainder)
- [x] Add CPU core detection (reads /proc/cpuinfo)

**Phase 6: Pass Data to Threads** ⏳ TODO
- [ ] Define thread argument struct in assembly
- [ ] Pack: loop_body_addr, start_idx, end_idx, data_ptr
- [ ] Pass struct pointer to clone()
- [ ] Thread unpacks and executes loop body

**Phase 7: Wait for Completion** ✅ COMPLETE
- [x] Add futex syscall wrapper (FutexWait, FutexWake)
- [x] Implement barrier: threads wait for each other
- [x] Each thread decrements counter atomically (sync/atomic)
- [x] Last thread wakes all waiting threads
- [x] Test: 4 goroutines synchronized at barrier

**Phase 8: Code Generation** ⏳ IN PROGRESS
- [x] Detect parallel loop in `compileLoopStatement()`
- [x] Validate constant range bounds
- [x] Calculate work distribution at compile time
- [x] Detailed diagnostic output
- [ ] Allocate shared memory for barrier and thread args
- [ ] Emit clone() calls in assembly
- [ ] Emit loop body execution per thread
- [ ] Emit futex wait barrier in assembly
- [ ] Emit cleanup code

**Testing**:
```flap
// Test 1: Simple parallel
4 @ i in 0..<100 { printf("%v\n", i) }

// Test 2: All cores
@@ item in data { process(item) }
```

**Files**: `lexer.go`, `ast.go`, `parser.go`, `parallel.go`

---

### 2. Hot Reload Polish
Infrastructure exists, just needs final wiring

**Step 1: Keep Process Alive**
- [ ] Modify `watchAndRecompile()` to not kill process
- [ ] Store process handle in variable
- [ ] Skip restart on successful hot reload

**Step 2: Detect Changed Functions**
- [ ] Use `IncrementalState.IncrementalRecompile()`
- [ ] Get list of changed hot function names
- [ ] Skip if no hot functions changed

**Step 3: Extract Machine Code**
- [ ] Compile changed functions to temp binary
- [ ] Call `ExtractFunctionCode()` for each changed func
- [ ] Get: function address, code bytes, length

**Step 4: Write to Shared Memory**
- [ ] Call `HotReloadManager.ReloadHotFunction()`
- [ ] Write new code bytes to mmap'd region
- [ ] Update function pointer in table

**Step 5: Atomic Swap**
- [ ] Use LOCK CMPXCHG to swap pointer
- [ ] Ensure running threads see new code
- [ ] Test: change physics value while running

**Test Case**:
```flap
hot physics = () => {
    gravity = 9.8  // Change this while running
}
```

**Files**: `main.go`

---

### 3. Networking Polish
Basic UDP works. Add quality-of-life features.

**Step 1: Check Return Values**
- [ ] After sendto(): check rax for errors
- [ ] After recvfrom(): check rax for errors
- [ ] Jump to error handler on failure

**Step 2: Error Messages**
- [ ] Print "Send failed: port %d" on ECONNREFUSED
- [ ] Print "Receive failed" with errno
- [ ] Continue loop instead of crashing

**Step 3: Bytes to String**
- [ ] Allocate string from received buffer
- [ ] Pass string length from rax (bytes received)
- [ ] Store in message variable

**Step 4: Extract Sender Info**
- [ ] Parse sockaddr_in.sin_addr (4 bytes)
- [ ] Parse sockaddr_in.sin_port (2 bytes)
- [ ] Format as "IP:port" string
- [ ] Store in sender variable

**Step 5: Connection Tracking (Optional)**
- [ ] Create hash map for sender addresses
- [ ] Track: last_seen timestamp per sender
- [ ] Timeout stale connections (60 seconds)
- [ ] Clean up hash map entries

**Test Case**:
```flap
@ msg, from in ":5000-5010" {
    printf("From %v: %v\n", from, msg)
}
```

**Files**: `parser.go`

---

## 📋 Optional Nice-to-Haves

### Atomic Operations
For thread-safe shared state in parallel loops

- [ ] Add `atomic_add(ptr, value)` builtin
- [ ] Use LOCK XADD instruction
- [ ] Add `atomic_cas(ptr, old, new)` builtin
- [ ] Use LOCK CMPXCHG instruction
- [ ] Add `mutex_lock(ptr)` builtin
- [ ] Use futex syscall
- [ ] Add `mutex_unlock(ptr)` builtin
- [ ] Test: increment shared counter from 4 threads

**Benefit**: Safe parallel loops with shared state

### Steamworks FFI
For shipping commercial games on Steam

- [ ] Parse C++ header files
- [ ] Handle name mangling (e.g., `_Z11SteamAPI_Initv`)
- [ ] Support callback function pointers
- [ ] Add achievement wrapper functions
- [ ] Add leaderboard wrappers
- [ ] Test: unlock achievement from Flap code

**Benefit**: Ship on Steam platform

---

## 🎯 1.6 Release Checklist

**Core Features:**
- [x] UDP networking (send/receive)
- [x] Port availability and fallback
- [x] Hot reload infrastructure
- [x] Spawn background processes
- [x] Tail call optimization
- [~] Parallel loops (Infrastructure complete, codegen pending)

**Quality:**
- [ ] Parallel loops: test with 10k items across 8 threads
- [ ] Hot reload: test changing physics constants live
- [ ] Networking: test 1000 messages/second throughput
- [ ] Clean VM: install and run on fresh Ubuntu 22.04
- [ ] Memory: run valgrind, fix any leaks

**Documentation:**
- [ ] Add parallel loop examples to README
- [ ] Write networking tutorial (client + server)
- [ ] Document hot reload workflow
- [ ] Add troubleshooting section

---

## 🚀 Post-1.6 Ideas

Deferred until after 1.6 ships:

- **ENet Protocol**: Reliable channels, packet ordering, ACKs
- **Trampolines**: Deep recursion without stack overflow
- **Macros**: Compile-time metaprogramming
- **CPS Transform**: All function calls become tail calls
- **Multiplatform**: Windows, macOS ARM, RISC-V
- **Python Syntax**: Colon + indentation alternative
- **Let Bindings**: Local recursive definitions

---

The core is already solid. Just need parallel loops + polish.
