# Flapc TODO - Version 1.6 Release

**Target**: x86-64 Linux game development
**Philosophy**: Minimal, elegant, implementable

---

## 🔥 Critical Path to 1.6

### 1. Parallel Loops (CPU Parallelism)
Essential for game performance: physics, AI, rendering

**Phase 1: Parser & AST** (1-2 hours)
- [ ] Add `@@` token to lexer (all cores syntax)
- [ ] Parse `N @` prefix (e.g., `4 @ item in data { }`)
- [ ] Add `NumThreads` field to `LoopStmt` in ast.go
- [ ] Handle `@@` as `NumThreads = 0` (detect cores at runtime)

**Phase 2: Clone Syscall** (1 hour)
- [ ] Add `clone()` syscall wrapper in new `parallel.go`
- [ ] Implement basic thread creation with CLONE_VM flag
- [ ] Test: spawn 4 threads, each prints thread ID

**Phase 3: Work Distribution** (2-3 hours)
- [ ] Calculate chunk size: `total_items / num_threads`
- [ ] Each thread processes: `[start_idx .. end_idx]`
- [ ] Pass loop body address + data range to threads
- [ ] Use futex for thread join (wait for completion)

**Phase 4: Code Generation** (2-3 hours)
- [ ] Generate parallel dispatch in `compileLoopStatement()`
- [ ] Allocate shared stack space for results
- [ ] Emit clone() calls with correct flags
- [ ] Implement barrier synchronization with futex

**Testing**:
```flap
// Test 1: Simple parallel
4 @ i in 0..<100 { printf("%v\n", i) }

// Test 2: All cores
@@ item in data { process(item) }
```

**Files**: `lexer.go`, `ast.go`, `parser.go`, `parallel.go`
**Estimate**: 6-9 hours total

---

### 2. Hot Reload Polish
Infrastructure exists, just needs final wiring

**Quick Wins** (2-3 hours):
- [ ] Modify `watchAndRecompile()`: keep process alive, don't restart
- [ ] On file change: compile only changed hot functions
- [ ] Extract machine code from new binary
- [ ] Write code to shared memory region (existing mmap)
- [ ] Update function pointer atomically

**Test Case**:
```flap
hot physics = () => {
    gravity = 9.8  // Change this while running
}
```

**Files**: `main.go` (50 lines of changes)
**Estimate**: 2-3 hours

---

### 3. Networking Polish
Basic UDP works. Add quality-of-life features.

**Phase 1: Error Handling** (1 hour)
- [ ] Check send/receive return values
- [ ] Print error message on failure
- [ ] Handle ECONNREFUSED gracefully

**Phase 2: String Conversion** (2 hours)
- [ ] Convert received bytes to string
- [ ] Pass actual message content to loop body
- [ ] Extract sender IP:port as string

**Phase 3: Connection Management** (Optional, 3-4 hours)
- [ ] Track active connections in hash map
- [ ] Detect disconnects (timeout)
- [ ] Clean up stale entries

**Test Case**:
```flap
@ msg, from in ":5000-5010" {
    printf("From %v: %v\n", from, msg)
}
```

**Files**: `parser.go` (100 lines)
**Estimate**: 3-7 hours depending on scope

---

## 📋 Optional Nice-to-Haves

### Atomic Operations (for parallel safety)
- [ ] Add `atomic_add(ptr, value)` builtin
- [ ] Add `atomic_cas(ptr, old, new)` builtin
- [ ] Add `mutex_lock(ptr)` / `mutex_unlock(ptr)` builtins
- [ ] Use futex syscall for implementation

**Estimate**: 3-4 hours
**Benefit**: Thread-safe shared state in parallel loops

### Steamworks FFI
- [ ] Parse C++ headers with name mangling
- [ ] Handle callback function pointers
- [ ] Add Steam API wrappers

**Estimate**: 6-8 hours
**Benefit**: Ship commercial games on Steam

---

## 🎯 1.6 Release Checklist

**Core Features Complete:**
- [x] UDP networking (send/receive)
- [x] Port availability and fallback
- [x] Hot reload infrastructure
- [x] Spawn background processes
- [x] Tail call optimization
- [ ] Parallel loops (in progress)

**Quality:**
- [ ] Test parallel loops with 10k+ items
- [ ] Test hot reload with physics loop
- [ ] Test UDP with 1000 messages/sec
- [ ] Run on clean Ubuntu 22.04 VM
- [ ] Memory leak check with valgrind

**Documentation:**
- [ ] Update README with parallel loop examples
- [ ] Add networking tutorial
- [ ] Document hot reload workflow

---

## 🚀 Post-1.6 Ideas

Deferred until after 1.6 ships:

- **ENet Protocol**: Reliable channels, fragmentation, ACKs
- **Trampolines**: Deep recursion without TCO
- **Macros**: Pattern-based metaprogramming
- **CPS Transform**: Internal tail call conversion
- **Multiplatform**: Windows, macOS ARM, RISC-V
- **Python Syntax Pack**: Colon + indentation
- **Let Bindings**: Local recursive definitions

---

## Summary

**To ship 1.6:**
1. Parallel loops (1-2 days)
2. Hot reload polish (half day)
3. Testing + docs (1 day)

**Total estimate**: 2-4 days of focused work

Everything else is optional or post-1.6. The core is already solid.
