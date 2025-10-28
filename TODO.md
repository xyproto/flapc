# Flapc TODO - Version 1.6 Release

**Target**: x86-64 Linux game development
**Philosophy**: Minimal, elegant, implementable

---

## 🔥 Critical Path to 1.6

### 1. Parallel Loops - V4 Complete, V5 In Progress

**Status**: Futex barrier synchronization working! Thread spawning, work distribution, and synchronization complete.

**✅ Completed (V1-V4):**
- Lexer & Parser: `@@` and `N @` syntax fully working
- AST: NumThreads field tracks parallelism level
- Thread spawning: clone() syscall with CLONE_VM
- Work distribution: GetThreadWorkRange() splits iterations
- CPU detection: Reads /proc/cpuinfo for core count
- Barrier synchronization: LOCK XADD + futex WAIT/WAKE
- New files: atomic.go (LOCK XADD), dec.go (DEC instruction)
- Verification: strace shows futex coordination working

**⏳ V5 - Full Loop Body Execution (Next):**
- [ ] Set up proper iterator variable in child thread stack frame
- [ ] Register iterator in fc.variables for child thread
- [ ] Convert iteration counter (int) to loop variable (float64)
- [ ] Execute actual loop body statements (compileStatements)
- [ ] Handle variable scoping in child threads
- [ ] Test: `@@ i in 0..<10 { printf("Loop: %v\n", i) }`

**📋 V6 - Multiple Threads (Future):**
- [ ] Spawn N child threads (currently spawns 1)
- [ ] Pass different work ranges to each thread
- [ ] Update barrier to count=N (currently count=1)
- [ ] All threads coordinate via single barrier
- [ ] Test: 4 threads with 100 iterations (25 each)

**📋 V7 - Dynamic Ranges (Future):**
- [ ] Support variable range bounds (currently constants only)
- [ ] Runtime work distribution calculation
- [ ] Pass range bounds via thread arguments

**Current Test:**
```bash
$ ./flapc test.flap -o test && ./test
0  # Child thread output
1
2
3
4
Done  # Parent continues after barrier
```

**Files**: `lexer.go`, `ast.go`, `parser.go`, `parallel.go`, `atomic.go`, `dec.go`

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
- [~] Parallel loops (V4 complete: barriers working, V5 pending: full loop bodies)

**Quality:**
- [x] Parallel loops: futex barrier synchronization verified with strace
- [ ] Parallel loops: test full loop body with printf
- [ ] Parallel loops: test with 10k items across N threads
- [ ] Hot reload: test changing physics constants live
- [ ] Networking: test 1000 messages/second throughput
- [ ] Clean VM: install and run on fresh Ubuntu 22.04
- [ ] Memory: run valgrind, fix any leaks

**Documentation:**
- [x] Update LANGUAGE.md with parallel loop implementation status
- [x] Update README.md with V4 progress
- [x] Add futex barrier learnings to LEARNINGS.md
- [x] Update TODO.md with V4 completion and V5 roadmap
- [ ] Add parallel loop code examples to testprograms/
- [ ] Write networking tutorial (client + server)
- [ ] Document hot reload workflow

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
