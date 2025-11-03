# ENet Networking Examples

These examples demonstrate Flapc's C FFI capabilities by interfacing with the ENet networking library.

## About ENet

ENet is a reliable UDP networking library designed for real-time applications and games. It's used by:
- Minecraft (Bedrock Edition)
- Star Citizen
- Many indie games

Features:
- Reliable and unreliable packet delivery
- Sequenced and ordered packet delivery
- Connection management
- Fragmentation and reassembly
- Low latency (UDP-based)

## Prerequisites

### Install ENet

**Ubuntu/Debian:**
```bash
sudo apt-get install libenet-dev
```

**Arch Linux:**
```bash
sudo pacman -S enet
```

**macOS:**
```bash
brew install enet
```

**From Source:**
```bash
git clone https://github.com/lsalzman/enet.git
cd enet
autoreconf -vfi
./configure
make
sudo make install
```

## Building

The examples require linking against libenet:

```bash
# Compile server
flapc server.flap -o server

# Compile client
flapc client.flap -o client
```

**Note:** If ENet is not in standard library paths, you may need to specify:
```bash
flapc server.flap -o server -L/usr/local/lib
```

## Running

### Terminal 1 - Start Server
```bash
./server
```

Expected output:
```
ENet initialized successfully
ENet server started on port 7777
Waiting for connections...
```

### Terminal 2 - Connect Client
```bash
./client
```

Expected output:
```
ENet initialized successfully
Connecting to 127.0.0.1:7777...
Connected to server!
Sent message to server
Disconnected from server
Client shutting down...
```

## Code Structure

### server.flap

Demonstrates:
- ENet initialization and cleanup (with `defer`)
- Creating a server host on port 7777
- Accepting up to 32 clients
- Event loop with 1-second timeout
- Handling connection, receive, and disconnect events
- Reading packet data
- Proper resource cleanup

Key Functions Used:
- `enet.enet_initialize()` - Initialize ENet library
- `enet.enet_host_create()` - Create server host
- `enet.enet_host_service()` - Process events
- `enet.enet_packet_destroy()` - Free packet memory
- `enet.enet_host_destroy()` - Destroy host
- `enet.enet_deinitialize()` - Shutdown ENet

### client.flap

Demonstrates:
- Creating a client host
- Connecting to a server
- Sending reliable packets
- Waiting for responses
- Graceful disconnect
- Timeout handling

Key Functions Used:
- `enet.enet_host_connect()` - Connect to server
- `enet.enet_packet_create()` - Create packet
- `enet.enet_peer_send()` - Send packet
- `enet.enet_host_flush()` - Flush send queue
- `enet.enet_peer_disconnect()` - Graceful disconnect

## C FFI Features Demonstrated

1. **C Library Imports**: `import enet as enet`
2. **Structure Access**: Reading/writing C struct fields with `read_u32`, `write_u16`, etc.
3. **Memory Allocation**: `alloc()` for creating structures
4. **Pointer Handling**: Passing pointers to C functions
5. **Defer Statement**: Automatic cleanup with `defer`
6. **String to C String**: `"text" as cstr` conversion

## Troubleshooting

### "Failed to initialize ENet"
- ENet library not installed or not in library path
- Try: `export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH`

### "undefined reference to enet_initialize"
- Compilation succeeded but linking failed
- Ensure `-lenet` is passed to linker (flapc does this automatically for imported libraries)

### Connection Timeout
- Server not running
- Firewall blocking port 7777
- Wrong IP address or port

## Advanced Usage

### Multiple Channels

ENet supports multiple channels per connection:
```flap
// Channel 0: reliable ordered (game state)
// Channel 1: unreliable (position updates)
peer := enet.enet_host_connect(client, server_addr, 2, 0)

// Send on channel 0 (reliable)
enet.enet_peer_send(peer, 0, reliable_packet)

// Send on channel 1 (unreliable)
enet.enet_peer_send(peer, 1, unreliable_packet)
```

### Packet Flags

- `1` - ENET_PACKET_FLAG_RELIABLE (guaranteed delivery)
- `2` - ENET_PACKET_FLAG_UNSEQUENCED (can arrive out of order)
- `4` - ENET_PACKET_FLAG_NO_ALLOCATE (don't copy data)
- `8` - ENET_PACKET_FLAG_UNRELIABLE_FRAGMENT (allow fragmentation)

### Bandwidth Limits

```flap
host := enet.enet_host_create(
    addr,
    32,           // peers
    2,            // channels
    57600 / 8,    // 56K incoming
    14400 / 8     // 14.4K outgoing
)
```

## Testing

The Go test suite includes compilation tests for these examples:

```bash
go test -run TestENetCompilation
```

This verifies that the examples compile successfully and demonstrates C FFI correctness.

## References

- [ENet Website](http://enet.bespin.org/)
- [ENet GitHub](https://github.com/lsalzman/enet)
- [ENet Tutorial](http://enet.bespin.org/Tutorial.html)
- [Flapc C FFI Guide](../../LANGUAGE.md#c-ffi)
