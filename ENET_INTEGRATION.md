# ENet Integration for Flap Concurrency

## Overview

**ENet** is Flap's primary concurrency mechanism. It provides:
- Fast, reliable UDP-based messaging (like TCP reliability + UDP speed)
- Unified syntax for IPC (local) and networking (remote)
- Simple send/receive model with no setup complexity

## Flap Syntax

### Port Literals
```flap
:5000                    // Port on localhost (ENet server)
"server.com:8080"        // Remote address (string)
```

### Sending Messages
```flap
:5000 <== "hello"                // Send to local port
"192.168.1.100:8080" <== "data"  // Send to remote address
```

### Receiving Messages
```flap
// Basic receive loop
@ msg, from in :5000 {
    println(f"Got: {msg} from {from}")
}

// With pattern matching
@ msg, from in :8080 {
    msg {
        "ping" -> from <== "pong"
        "quit" -> ret
        ~> from <== "unknown"
    }
}

// With iteration limit
@ msg, from in :5000 max 100 {
    process(msg)
}
```

## Implementation Status

### Current Status (as of 2025-11-13)
- ✅ AST nodes exist: `SendExpr`, `ReceiveLoopStmt`
- ✅ LANGUAGE.md documented
- ✅ Grammar rules defined
- ❌ ENet tests failing (need investigation)
- ❌ Codegen may be incomplete

### What Needs to Work

1. **Parse `:5000` as port literal**
   - Token: `TOKEN_COLON` followed by `TOKEN_NUMBER`
   - AST: `NumberExpr` with special marker, or `PortLiteralExpr`

2. **Parse `target <== message`**
   - Token: `TOKEN_SEND` or `TOKEN_LESS_EQUAL_EQUAL`
   - AST: `SendExpr{Target, Message}`

3. **Parse `@ msg, from in :5000 { ... }`**
   - Already in parser (check for bugs)
   - AST: `ReceiveLoopStmt`

4. **Codegen: Link with ENet**
   - Option A: Bundle ENet object code (libenet.a)
   - Option B: Use header-only ENet port
   - Option C: Generate ENet calls using TCC/GCC

### Implementation Strategy

#### Phase 1: Use C ENet Library (Simplest)
```go
// In codegen.go
func (fc *FlapCompiler) compileReceiveLoop(loop *ReceiveLoopStmt) {
    // Call C library functions:
    // enet_initialize()
    // enet_host_create()
    // enet_host_service() in loop
    // Extract msg and sender address
}

func (fc *FlapCompiler) compileSendExpr(send *SendExpr) {
    // Parse target (":5000" or "host:port")
    // Call enet_host_connect() if needed
    // Call enet_peer_send()
}
```

Link with: `-lenet` or bundle `libenet.a`

#### Phase 2: Header-Only ENet (Future)
- Port ENet to Flap's inline assembly style
- Emit machine code directly (no linking)
- Requires significant work but cleaner

#### Phase 3: TCC Helper (Pragmatic)
```go
// Generate C code for ENet operations
c_code := `
    #include <enet/enet.h>
    void flap_send(const char* target, const char* msg) {
        // ENet send implementation
    }
`
// Compile with TCC at runtime or build time
// Link into executable
```

## Testing Plan

### Unit Tests (Minimal)
```bash
# Test 1: Port literal parsing
go test -v -run="TestPortLiteral"

# Test 2: Send operator parsing  
go test -v -run="TestSendExpr"

# Test 3: Receive loop parsing
go test -v -run="TestReceiveLoop"
```

### Integration Tests
```bash
# Test 4: Echo server
go test -v -run="TestENetEcho"

# Test 5: Multiple clients
go test -v -run="TestENetMultiClient"
```

### Minimal Working Example
```flap
// server.flap
@ msg, from in :8080 {
    println(f"Received: {msg}")
    from <== f"Echo: {msg}"
}
```

```flap
// client.flap
:8080 <== "hello"
@ reply, from in :9000 max 1 {
    println(f"Got: {reply}")
}
```

## Current Test Failures

From TODO.md:
- `TestENetCompilation/enet_simple` - FAILING
- `TestENetCodeGeneration/simple_test.flap` - FAILING

**Action:** Run tests with `-v` to see actual errors:
```bash
go test -v -run="TestENet"
```

## Dependencies

### ENet Library
- **Source:** http://enet.bespin.org/
- **License:** MIT
- **Size:** ~30KB source
- **Platforms:** Linux, macOS, Windows

### Installation
```bash
# Ubuntu/Debian
sudo apt install libenet-dev

# macOS
brew install enet

# From source
git clone https://github.com/lsalzman/enet
cd enet
./configure
make && sudo make install
```

## Design Decisions

### Why ENet over alternatives?

| Feature | ENet | ZeroMQ | Raw UDP | Raw TCP |
|---------|------|--------|---------|---------|
| Reliability | ✅ | ✅ | ❌ | ✅ |
| Low latency | ✅ | ⚠️ | ✅ | ❌ |
| Simple API | ✅ | ⚠️ | ⚠️ | ⚠️ |
| Small footprint | ✅ | ❌ | ✅ | ✅ |
| No dependencies | ✅ | ❌ | ✅ | ✅ |

ENet = TCP reliability + UDP speed + simple API

### Why Unified IPC/Network Syntax?

**Traditional approach (complex):**
```c
// IPC
int fd = unix_socket(...);
// vs Network  
int fd = tcp_socket(...);
```

**Flap approach (simple):**
```flap
":5000" <== msg        // IPC (localhost)
"host:5000" <== msg    // Network (remote)
```

Same syntax, automatic optimization.

## Next Steps

1. **Fix failing tests** (see TODO.md)
2. **Implement codegen** for ENet operations
3. **Bundle libenet.a** with compiler
4. **Document C FFI approach** for ENet
5. **Add more examples** to test suite

## References

- ENet website: http://enet.bespin.org/
- ENet tutorial: http://enet.bespin.org/Tutorial.html
- Flap AST: `ast.go` lines 225-237, 742-750
- Flap grammar: `LANGUAGE.md` receive_loop, send_expr

---

**Status:** Documentation complete, implementation in progress  
**Priority:** HIGH - Core concurrency mechanism  
**Blocker:** Test failures need investigation
