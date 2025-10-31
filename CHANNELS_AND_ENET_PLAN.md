# Channels and ENet Networking Plan for Flapc

## Overview

This document proposes two major features for Flapc:
1. **Channels** - Go-style CSP (Communicating Sequential Processes) for thread communication
2. **ENet Integration** - High-level networking for multiplayer games

Both features are designed to be **zero-runtime** (no garbage collector), **type-safe**, and **game-dev friendly**.

---

# Part 1: Channels (CSP-style Concurrency)

## Motivation

Current parallel loops (`@@`) are great for data parallelism, but lack communication primitives. Channels provide:
- **Safe inter-thread communication**
- **Producer-consumer patterns**
- **Pipeline architectures**
- **Work distribution**

---

## Proposed Syntax

### 1. Basic Channel Creation

```flap
// Create unbuffered channel (blocking send/receive):
ch := chan()

// Create buffered channel (capacity 10):
ch := chan(10)

// Typed channels (future enhancement):
ch := chan<int32>()
```

**Implementation:**
- Channels are pointers to heap-allocated structures
- Backed by ring buffer + mutex + condition variables
- Syscalls: `futex` for Linux

---

### 2. Send and Receive Operations

```flap
// Send value to channel:
ch <- 42

// Receive value from channel:
value := <-ch

// Non-blocking receive with timeout (future):
value := <-ch ?? default_value
```

**Semantics:**
- `ch <- value`: Block until space available
- `value := <-ch`: Block until value available
- Close detection: Receive returns special value (e.g., `0`) when closed

---

### 3. Channel Closing

```flap
close(ch)

// After close:
// - Send panics
// - Receive returns 0 (or special marker)
// - Multiple receives safe (idempotent)
```

---

### 4. Select Statement (Multiplexing)

```flap
select {
    value := <-ch1 -> {
        println("Received from ch1")
    }
    value := <-ch2 -> {
        println("Received from ch2")
    }
    timeout(1000) -> {  // milliseconds
        println("Timeout!")
    }
}
```

**Implementation:**
- Poll all channels simultaneously
- Use `epoll` or `futex` for efficient waiting
- Random selection if multiple ready

---

### 5. Producer-Consumer Example

```flap
// Producer goroutine-style syntax:
spawn {
    @ i in 0..<100 {
        ch <- i
    }
    close(ch)
}

// Consumer:
@ {
    value := <-ch
    value == 0 { break }  // Channel closed
    process(value)
}
```

---

### 6. Buffered Channels for Performance

```flap
results := chan(1000)  // Buffer up to 1000 results

// Workers send results:
@@ worker_id in 0..<4 {
    result := do_work(worker_id)
    results <- result
}

// Main thread collects:
@ i in 0..<4 {
    r := <-results
    println(r)
}
```

---

## Channel Implementation Details

### Data Structure:

```flap
cstruct Channel {
    buffer as ptr       // Ring buffer for values
    capacity as int32   // Buffer size (0 = unbuffered)
    size as int32       // Current element count
    head as int32       // Read position
    tail as int32       // Write position
    lock as ptr         // Mutex (futex-based)
    send_wait as ptr    // Condition variable for senders
    recv_wait as ptr    // Condition variable for receivers
    closed as int32     // Close flag
}
```

### Operations:

```flap
// Send (blocking):
chan_send := (ch as ptr, value as int64) -> {
    lock(ch.lock)
    defer unlock(ch.lock)

    // Wait for space:
    ch.size == ch.capacity {
        wait(ch.send_wait, ch.lock)
    }

    // Write to buffer:
    write_i64(ch.buffer, ch.tail * 8, value)
    ch.tail = (ch.tail + 1) % ch.capacity
    ch.size = ch.size + 1

    // Wake receiver:
    signal(ch.recv_wait)
}

// Receive (blocking):
chan_recv := (ch as ptr) -> int64 {
    lock(ch.lock)
    defer unlock(ch.lock)

    // Wait for value:
    ch.size == 0 and ch.closed == 0 {
        wait(ch.recv_wait, ch.lock)
    }

    // Channel closed and empty:
    ch.closed == 1 and ch.size == 0 {
        -> 0  // Sentinel value
    }

    // Read from buffer:
    value := read_i64(ch.buffer, ch.head * 8)
    ch.head = (ch.head + 1) % ch.capacity
    ch.size = ch.size - 1

    // Wake sender:
    signal(ch.send_wait)

    -> value
}
```

---

## Spawn Syntax for Goroutines

```flap
// Spawn lightweight thread:
spawn {
    // This runs in separate thread
    println("Hello from thread!")
}

// Spawn with arguments:
spawn (x) {
    process(x)
}(42)

// Spawn multiple:
@ i in 0..<10 {
    spawn (id) {
        worker(id)
    }(i)
}
```

**Implementation:**
- Use existing parallel loop infrastructure
- `spawn` is syntactic sugar for single-iteration `@@` loop
- Channels provide communication

---

## Pipeline Pattern Example

```flap
main := () -> {
    // Stage 1: Generate numbers
    numbers := chan(10)
    spawn {
        @ i in 0..<100 {
            numbers <- i
        }
        close(numbers)
    }

    // Stage 2: Square numbers
    squares := chan(10)
    spawn {
        @ {
            n := <-numbers
            n == 0 { break }  // Closed
            squares <- n * n
        }
        close(squares)
    }

    // Stage 3: Print results
    @ {
        s := <-squares
        s == 0 { break }
        println(s)
    }
}
```

---

# Part 2: ENet Integration (Game Networking)

## Motivation

Building multiplayer games requires:
- **Reliable and unreliable messaging**
- **Connection management**
- **Bandwidth throttling**
- **Packet sequencing**

ENet is a proven library used by games like:
- Cube 2: Sauerbraten
- Wesnoth
- Many indie games

---

## Proposed ENet Syntax

### 1. Import ENet Library

```flap
import enet as enet
```

**Implementation:**
- Use existing C FFI
- Extract constants from DWARF debug info
- Auto-generate bindings

---

### 2. Initialize ENet

```flap
enet.enet_initialize()
defer enet.enet_deinitialize()
```

---

### 3. Create Server

```flap
// Create server address:
addr_ptr := alloc(enet.ENetAddress_SIZEOF)
enet.enet_address_set_host(addr_ptr, "0.0.0.0" as cstr)
write_u16(addr_ptr, enet.ENetAddress_port_OFFSET as int32, 7777)

// Create host:
host := enet.enet_host_create(
    addr_ptr,            // address
    32 as uint64,        // max clients
    2 as uint64,         // channel count
    0 as uint32,         // incoming bandwidth
    0 as uint32          // outgoing bandwidth
)

defer enet.enet_host_destroy(host)
```

---

### 4. Server Event Loop

```flap
event_ptr := alloc(enet.ENetEvent_SIZEOF)

@ {
    // Poll for events (timeout: 1000ms)
    result := enet.enet_host_service(host, event_ptr, 1000 as uint32)

    result > 0 {
        event_type := read_u32(event_ptr, enet.ENetEvent_type_OFFSET as int32)

        event_type == enet.ENET_EVENT_TYPE_CONNECT {
            println("Client connected!")
        }

        event_type == enet.ENET_EVENT_TYPE_RECEIVE {
            packet := read_ptr(event_ptr, enet.ENetEvent_packet_OFFSET as int32)
            data := read_ptr(packet, enet.ENetPacket_data_OFFSET as int32)
            length := read_u64(packet, enet.ENetPacket_dataLength_OFFSET as int32)

            println(f"Received {length} bytes")

            // Process data...

            // Destroy packet:
            enet.enet_packet_destroy(packet)
        }

        event_type == enet.ENET_EVENT_TYPE_DISCONNECT {
            println("Client disconnected")
        }
    }
}
```

---

### 5. Client Connection

```flap
// Create client (no server address):
client := enet.enet_host_create(
    0 as ptr,            // no server
    1 as uint64,         // 1 outgoing connection
    2 as uint64,         // 2 channels
    0 as uint32,         // unlimited bandwidth
    0 as uint32
)

// Server address:
server_addr := alloc(enet.ENetAddress_SIZEOF)
enet.enet_address_set_host(server_addr, "localhost" as cstr)
write_u16(server_addr, enet.ENetAddress_port_OFFSET as int32, 7777)

// Connect:
peer := enet.enet_host_connect(client, server_addr, 2 as uint64, 0 as uint32)

// Wait for connection:
connected := 0
@ connected == 0 {
    event_ptr := alloc(enet.ENetEvent_SIZEOF)
    result := enet.enet_host_service(client, event_ptr, 5000 as uint32)

    result > 0 {
        event_type := read_u32(event_ptr, enet.ENetEvent_type_OFFSET as int32)
        event_type == enet.ENET_EVENT_TYPE_CONNECT {
            println("Connected to server!")
            connected = 1
        }
    }
}
```

---

### 6. Send Reliable Packet

```flap
// Create packet:
data := "Hello, server!" as cstr
packet := enet.enet_packet_create(
    data as ptr,
    14 as uint64,                         // length
    enet.ENET_PACKET_FLAG_RELIABLE as uint32
)

// Send to peer:
enet.enet_peer_send(peer, 0 as uint8, packet)

// Flush (send immediately):
enet.enet_host_flush(client)
```

---

### 7. Send Unreliable Packet (for fast updates)

```flap
// Game state update (position):
pos_data := alloc(12)
write_f32(pos_data, 0, player_x)
write_f32(pos_data, 4, player_y)
write_f32(pos_data, 8, player_z)

packet := enet.enet_packet_create(
    pos_data as ptr,
    12 as uint64,
    enet.ENET_PACKET_FLAG_UNSEQUENCED as uint32  // Fast, no guarantee
)

enet.enet_peer_send(peer, 1 as uint8, packet)  // Channel 1
```

---

## High-Level ENet Wrapper (Future)

### Simplified API:

```flap
// Create server with high-level API:
server := enet_server(7777, 32)  // port, max clients

// Event handler:
server.on_connect = (client_id as int32) -> {
    println(f"Client {client_id} connected")
}

server.on_receive = (client_id as int32, data as ptr, length as int32) -> {
    println(f"Received {length} bytes from {client_id}")
}

server.on_disconnect = (client_id as int32) -> {
    println(f"Client {client_id} disconnected")
}

// Run server:
@ {
    server.poll(1000)  // 1s timeout
}
```

---

## Combining Channels + ENet

### Game Server with Worker Threads:

```flap
import enet as enet

main := () -> {
    // Initialize ENet:
    enet.enet_initialize()
    defer enet.enet_deinitialize()

    // Create channels:
    incoming := chan(1000)  // Client messages
    outgoing := chan(1000)  // Server responses

    // Network thread:
    spawn {
        host := create_server(7777)
        @ {
            event := poll_enet(host, 100)
            event.type == CONNECT {
                println("Client connected")
            }
            event.type == RECEIVE {
                incoming <- event.data  // Send to game logic
            }
        }
    }

    // Game logic thread:
    spawn {
        @ {
            msg := <-incoming
            response := process_game_logic(msg)
            outgoing <- response
        }
    }

    // Send thread:
    spawn {
        @ {
            response := <-outgoing
            send_to_clients(response)
        }
    }

    // Keep main thread alive:
    @ { sleep(1000) }
}
```

---

## Implementation Roadmap

### Phase 1: Channels MVP
- ✅ Existing parallel loop infrastructure (threads, barriers)
- ⏳ Implement `chan()` builtin (allocate channel structure)
- ⏳ Implement `<-` send operator
- ⏳ Implement `<-` receive operator
- ⏳ Implement `close()` builtin
- ⏳ Add futex-based mutex/condvar

**Estimated effort:** 2-3 weeks

### Phase 2: Spawn Syntax
- ⏳ Add `spawn { ... }` syntax
- ⏳ Compile to single-iteration parallel loop
- ⏳ Integrate with channel operations

**Estimated effort:** 1 week

### Phase 3: Select Statement
- ⏳ Add `select { }` syntax
- ⏳ Implement channel polling
- ⏳ Add timeout support

**Estimated effort:** 2 weeks

### Phase 4: ENet Integration
- ⏳ Import ENet library via FFI
- ⏳ Extract constants automatically
- ⏳ Create example programs (server/client)
- ⏳ Document patterns

**Estimated effort:** 1 week (FFI already works)

### Phase 5: High-Level ENet Wrapper
- ⏳ Create `enet_server()` and `enet_client()` helpers
- ⏳ Callback-based API
- ⏳ Automatic packet management

**Estimated effort:** 1-2 weeks

---

## Memory Management Considerations

### Channels:
- Allocated via `malloc` (or custom allocator)
- Freed explicitly with `free()` or via `defer`
- No garbage collector needed

### ENet:
- ENet manages its own memory
- Packets freed with `enet_packet_destroy()`
- Hosts freed with `enet_host_destroy()`

### RAII Pattern with Defer:
```flap
ch := chan(10)
defer free_channel(ch)

host := enet_host_create(...)
defer enet_host_destroy(host)

// Automatic cleanup on scope exit
```

---

## Safety Considerations

### Channel Safety:
- **Data races:** Prevented by mutex (futex)
- **Deadlocks:** Possible (user responsibility)
- **Use-after-close:** Receive returns sentinel (0)
- **Send-after-close:** Could panic or return error

### ENet Safety:
- **Buffer overruns:** ENet handles internally
- **Connection drops:** Event-driven (DISCONNECT)
- **Invalid packets:** ENet validates checksums

---

## Performance Characteristics

### Channels:
- **Send/Receive:** ~50-100ns (futex wake)
- **Buffered send:** ~20ns (no wake needed)
- **Select overhead:** ~100ns per channel

### ENet:
- **Latency:** ~1-5ms (LAN), ~20-100ms (WAN)
- **Throughput:** ~10-100 MB/s (depends on network)
- **CPU overhead:** ~1-5% per 1000 packets/sec

---

## Example: Multiplayer Game

```flap
import enet as enet

main := () -> {
    enet.enet_initialize()
    defer enet.enet_deinitialize()

    // Create server:
    host := create_server(7777)
    defer enet_host_destroy(host)

    // Game state:
    players := alloc(100 * 16)  // 100 players, 16 bytes each

    // Game loop:
    @ {
        // Network events:
        event := poll_enet(host, 16)  // 16ms = 60 FPS

        event.type == CONNECT {
            player_id := assign_player_slot()
            println(f"Player {player_id} joined")
        }

        event.type == RECEIVE {
            player_id := event.peer_id
            data := event.data

            // Parse input (e.g., movement):
            dx := read_f32(data, 0)
            dy := read_f32(data, 4)

            // Update player position:
            update_player(players, player_id, dx, dy)
        }

        event.type == DISCONNECT {
            remove_player(event.peer_id)
        }

        // Update game logic:
        update_physics(players)

        // Broadcast state to all clients:
        @@ i in 0..<player_count {
            packet := create_state_packet(players, i)
            enet_peer_send(get_peer(i), 0 as uint8, packet)
        }

        enet_host_flush(host)
    }
}
```

---

## Testing Strategy

### Channel Tests:
- Send/receive basic values
- Buffered channel behavior
- Close detection
- Concurrent stress tests (1000 threads)
- Deadlock detection (timeout-based)

### ENet Tests:
- Server/client connection
- Reliable packet delivery
- Unreliable packet behavior
- Disconnection handling
- Bandwidth throttling

---

## Documentation Deliverables

1. **CHANNELS.md** - Channel usage guide
2. **ENET_GUIDE.md** - ENet integration tutorial
3. **MULTIPLAYER_PATTERNS.md** - Common game networking patterns
4. **CSP_EXAMPLES.md** - Concurrency pattern examples

---

## Alternatives Considered

### Why Channels over Shared Memory?
- **Safer:** No manual locking
- **Clearer:** Explicit communication
- **Easier:** Less prone to data races

### Why ENet over Raw Sockets?
- **Proven:** Used in production games
- **Feature-rich:** Reliability, sequencing, throttling
- **Cross-platform:** Works on Linux, Windows, macOS

### Why Not ZeroMQ?
- **Heavyweight:** Requires runtime
- **Overkill:** Games need simple request/response
- **ENet optimized:** For real-time, low-latency

---

## Conclusion

Adding **Channels** and **ENet** to Flapc would make it a **complete game development language**:

- ✅ **Concurrency:** Channels for safe inter-thread communication
- ✅ **Networking:** ENet for multiplayer games
- ✅ **Performance:** Zero-runtime overhead
- ✅ **Safety:** Type-safe, no garbage collector
- ✅ **Proven:** Based on Go (channels) and ENet (used in 100+ games)

**Estimated total effort:** 8-10 weeks for full implementation.

---

## Next Steps

1. ⏳ Implement channel MVP (futex-based)
2. ⏳ Add spawn syntax
3. ⏳ Test with simple producer-consumer
4. ⏳ Import ENet via FFI
5. ⏳ Create example multiplayer game
6. ⏳ Document patterns and best practices
