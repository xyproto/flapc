# Flapc TODO - Version 1.6 Release

**Target**: x86-64 Linux game development
**Philosophy**: Minimal, elegant, implementable

---

## 📊 Recent Progress (Session Update)

**Date**: 2025-10-28
**Milestone**: V4 Parallel Loops Complete

**Completed This Session:**
- ✅ V4 futex barrier synchronization (atomic.go, dec.go, parser.go)
- ✅ LOCK XADD atomic operations for x86-64/ARM64/RISC-V
- ✅ DEC instruction for all architectures
- ✅ Thread spawning with mmap + clone() syscalls
- ✅ Parent-child synchronization verified with strace
- ✅ Documentation: LANGUAGE.md, README.md, LEARNINGS.md updated
- ✅ Pushed 21 commits to origin/main

**Files Changed:**
- New: `atomic.go` (87 lines), `dec.go` (115 lines)
- Modified: `parser.go` (~300 lines for V4)
- Docs: 4 files updated (~250 lines)

**Total Impact**: 6 files, 544 insertions, 97 deletions

---

## 🔥 Critical Path to 1.6

### 1. Parallel Loops - V6 Complete, V5 In Progress

**Status**: V6 complete! N threads spawn, distribute work, coordinate via futex barriers.

**✅ Completed (V1-V6):**
- Lexer & Parser: `@@` and `N @` syntax fully working
- AST: NumThreads field tracks parallelism level
- Thread spawning: clone() syscall with CLONE_VM (N threads)
- Work distribution: GetThreadWorkRange() splits iterations across N threads
- CPU detection: Reads /proc/cpuinfo for core count
- Barrier synchronization: LOCK XADD + futex WAIT/WAKE (N+1 participants)
- Optimizer: Skip loop unrolling for parallel loops
- Verification: strace shows N clone() calls, futex coordination
- Testing: 2 threads x 5 iters, 4 threads x 20 iters, 2 threads x 100 iters

**✅ V5 - Full Loop Body Execution (COMPLETE!):**

V5 is now complete! Parallel loops can execute arbitrary loop bodies with local variables.

**Completed:**
- [x] Step 1: Removed hardcoded loop body
- [x] Step 2: Set up iterator variable on child stack (cvtsi2sd, store at rbp-16)
- [x] Step 3: Fixed variable context (use existing fc.variables from collectSymbols)
- [x] Step 4: Compile actual loop body statements
- [x] Step 5: Test with iterator-only loop (`@@ i in 0..<3 { y := i }`)
- [x] Step 6: Test with arithmetic (`@@ i in 0..<5 { x := i * 2; y := x + 10 }`)
- [x] Fixed V6 barrier race condition (parent now participates as (N+1)th thread)

**Key Learnings:**
- Two-pass compilation: collectSymbols registers variables, compileStatement generates code
- Loop-local variables are pre-registered during collectSymbols phase
- Must preserve fc.variables from collectSymbols, only override iterator offset
- Parent must participate in barrier to avoid lost wakeup race condition

**Tests Passing:**
- Empty loop body: `@@ i in 0..<3 { }`
- Simple assignment: `@@ i in 0..<3 { y := i }`
- Arithmetic: `@@ i in 0..<5 { x := i * 2; y := x + 10 }`
- Multiple threads: `4 @ i in 0..<20 { x := i + 48 }`
- External variables: `a := 100; @@ i in 0..<3 { x := i + a }` ✓

**Current Limitations:**
- Printf/function calls: Require proper calling conventions or position-independent code
- Shared mutable state: No atomic operations yet for thread-safe updates (Step 8)

**Next Steps (Optional):**
- [x] Step 7: External variable access (COMPLETE! Child threads can now read parent variables via r11)
  - collectLoopLocalVars() distinguishes parent vs loop-local variables
  - Variable reads/writes use r11 for parent vars, rbp for local vars
  - Tests passing: `a := 100; @@ i in 0..<3 { x := i + a }` ✓
- [ ] Step 8: Atomic operations for shared state (LOCK XADD, CMPXCHG)
- [ ] Step 9: Printf support (via wrapper or PIC)

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

**✅ Phase 1: Smart Restart (COMPLETE)**
- [x] Detect changed functions via `IncrementalState.IncrementalRecompile()`
- [x] Check if any changed functions are hot functions
- [x] Keep process alive if no hot functions changed
- [x] Only restart when hot functions actually change
- [x] Clear feedback messages

**Benefits**: Faster iteration on non-hot code, foundation for true hot reload

**📋 Phase 2: True Hot Reload (Future)**
Infrastructure exists, needs IPC wiring:

**Step 1: Shared Memory Setup**
- [ ] Game process creates mmap'd region for code
- [ ] Game process exposes function pointer table
- [ ] Compiler connects to shared memory

**Step 2: Detect Changed Functions**
- [x] Already working via `IncrementalState.IncrementalRecompile()`

**Step 3: Extract Machine Code**
- [ ] Call `ExtractFunctionCode()` for each changed hot function
- [ ] Get function address, code bytes, length

**Step 4: Write to Shared Memory via IPC**
- [ ] Compiler signals game process (e.g., via Unix socket)
- [ ] Game process receives new code bytes
- [ ] Call `HotReloadManager.LoadHotFunction()`
- [ ] Copy code to mmap'd executable page

**Step 5: Atomic Pointer Update**
- [ ] Update function pointer table atomically
- [ ] Use memory barrier to ensure visibility
- [ ] Test: change getValue() from 42 -> 100 while running

**Current Approach**: Smart restart (Phase 1 complete)
**Future Enhancement**: IPC-based hot patching (Phase 2)

**Files**: `main.go`, `hotreload.go`, `incremental.go`

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
