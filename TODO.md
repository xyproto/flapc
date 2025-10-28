# Flapc TODO - Version 1.6 Release

**Target**: x86-64 Linux game development
**Philosophy**: Minimal, elegant, implementable

---

## 🔥 Critical Path to 1.6

### 1. Parallel Loops (CPU Parallelism)
Essential for game performance: physics, AI, rendering

**Phase 1: Lexer & Parser**
- [ ] Add `@@` token to lexer (all cores syntax)
- [ ] Parse `N @` prefix (e.g., `4 @ item in data { }`)
- [ ] Handle `@@` as special case (detect cores at runtime)

**Phase 2: AST Changes**
- [ ] Add `NumThreads` field to `LoopStmt` in ast.go
- [ ] Update `String()` method for parallel loops
- [ ] Add `IsParallel()` helper method

**Phase 3: Basic Thread Creation**
- [ ] Create new `parallel.go` file
- [ ] Add `clone()` syscall wrapper
- [ ] Implement thread spawn with CLONE_VM flag
- [ ] Test: spawn single thread, print "Hello from thread"

**Phase 4: Thread ID & Verification**
- [ ] Add syscall to get thread ID
- [ ] Test: spawn 4 threads, each prints its TID
- [ ] Verify all threads run independently

**Phase 5: Work Distribution Math**
- [ ] Calculate: `chunk_size = total_items / num_threads`
- [ ] Calculate: `start_idx = thread_id * chunk_size`
- [ ] Calculate: `end_idx = start_idx + chunk_size`
- [ ] Handle remainder items (give to last thread)

**Phase 6: Pass Data to Threads**
- [ ] Define thread argument struct in assembly
- [ ] Pack: loop_body_addr, start_idx, end_idx, data_ptr
- [ ] Pass struct pointer to clone()
- [ ] Thread unpacks and executes loop body

**Phase 7: Wait for Completion**
- [ ] Add futex syscall wrapper
- [ ] Implement barrier: main thread waits for workers
- [ ] Each worker decrements counter atomically
- [ ] Main thread wakes when counter == 0

**Phase 8: Code Generation**
- [ ] Detect parallel loop in `compileLoopStatement()`
- [ ] Allocate shared memory for thread args
- [ ] Emit clone() calls in loop
- [ ] Emit futex wait barrier
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
- [ ] Parallel loops

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
