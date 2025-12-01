package main

// Self-extracting executable generation
// This provides compression for Flap executables to reduce size for demoscene use

// NOTE: Full self-extraction with decompressor stub is complex and requires:
// 1. Tiny decompressor stub (~150 bytes)
// 2. mmap() call to allocate executable memory
// 3. Decompression at runtime
// 4. Jump to decompressed code
//
// This is deferred for now. Instead, we focus on optimizing the existing executable format.
