// C helper functions for memory operations
// These are used by the Flap ENet examples to manipulate C structs

#include <stdint.h>
#include <stdlib.h>

// Write 32-bit unsigned integer at offset
void write_u32(void* ptr, size_t offset, uint32_t value) {
    *((uint32_t*)((char*)ptr + offset)) = value;
}

// Write 16-bit unsigned integer at offset
void write_u16(void* ptr, size_t offset, uint16_t value) {
    *((uint16_t*)((char*)ptr + offset)) = value;
}

// Read 32-bit unsigned integer from offset
uint32_t read_u32(void* ptr, size_t offset) {
    return *((uint32_t*)((char*)ptr + offset));
}

// Read 64-bit unsigned integer from offset
uint64_t read_u64(void* ptr, size_t offset) {
    return *((uint64_t*)((char*)ptr + offset));
}

// Read pointer from offset
void* read_ptr(void* ptr, size_t offset) {
    return *((void**)((char*)ptr + offset));
}
