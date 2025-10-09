package main

import (
	"fmt"
	"hash/fnv"
)

// FlapHashMap represents a hash map from uint64 to float64
// This is the fundamental datastructure in Flap
// All values (ints, strings, etc) are stored in such a hash map
type FlapHashMap struct {
	buckets []FlapHashBucket
	size    int
	count   int
}

type FlapHashBucket struct {
	key      uint64
	value    float64
	occupied bool
	next     *FlapHashBucket
}

// NewFlapHashMap creates a new hash map with the given initial size
func NewFlapHashMap(initialSize int) *FlapHashMap {
	if initialSize < 16 {
		initialSize = 16
	}
	return &FlapHashMap{
		buckets: make([]FlapHashBucket, initialSize),
		size:    initialSize,
		count:   0,
	}
}

// hash computes the hash of a uint64 key
func (m *FlapHashMap) hash(key uint64) uint64 {
	h := fnv.New64a()
	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		bytes[i] = byte(key >> (i * 8))
	}
	h.Write(bytes)
	return h.Sum64()
}

// Get retrieves a value from the hash map
func (m *FlapHashMap) Get(key uint64) (float64, bool) {
	idx := m.hash(key) % uint64(m.size)
	bucket := &m.buckets[idx]

	// Check the first bucket
	if bucket.occupied && bucket.key == key {
		return bucket.value, true
	}

	// Check the chain
	current := bucket.next
	for current != nil {
		if current.key == key {
			return current.value, true
		}
		current = current.next
	}

	return 0.0, false
}

// Set stores a value in the hash map
func (m *FlapHashMap) Set(key uint64, value float64) {
	idx := m.hash(key) % uint64(m.size)
	bucket := &m.buckets[idx]

	// If bucket is empty, use it
	if !bucket.occupied {
		bucket.key = key
		bucket.value = value
		bucket.occupied = true
		m.count++
		return
	}

	// If this key already exists in the first bucket, update it
	if bucket.key == key {
		bucket.value = value
		return
	}

	// Check the chain for existing key
	current := bucket.next
	prev := bucket
	for current != nil {
		if current.key == key {
			current.value = value
			return
		}
		prev = current
		current = current.next
	}

	// Add new entry to chain
	newBucket := &FlapHashBucket{
		key:      key,
		value:    value,
		occupied: true,
		next:     nil,
	}
	prev.next = newBucket
	m.count++

	// Check load factor and resize if needed
	if float64(m.count)/float64(m.size) > 0.75 {
		m.resize()
	}
}

// resize doubles the size of the hash map and rehashes all entries
func (m *FlapHashMap) resize() {
	oldBuckets := m.buckets
	m.size *= 2
	m.buckets = make([]FlapHashBucket, m.size)
	m.count = 0

	// Rehash all entries
	for i := range oldBuckets {
		bucket := &oldBuckets[i]
		if bucket.occupied {
			m.Set(bucket.key, bucket.value)
		}

		current := bucket.next
		for current != nil {
			m.Set(current.key, current.value)
			current = current.next
		}
	}
}

// Delete removes a key from the hash map
func (m *FlapHashMap) Delete(key uint64) bool {
	idx := m.hash(key) % uint64(m.size)
	bucket := &m.buckets[idx]

	// Check first bucket
	if bucket.occupied && bucket.key == key {
		if bucket.next != nil {
			// Move next bucket to first position
			*bucket = *bucket.next
		} else {
			// Clear the bucket
			bucket.key = 0
			bucket.value = 0.0
			bucket.occupied = false
			bucket.next = nil
		}
		m.count--
		return true
	}

	// Check chain
	prev := bucket
	current := bucket.next
	for current != nil {
		if current.key == key {
			prev.next = current.next
			m.count--
			return true
		}
		prev = current
		current = current.next
	}

	return false
}

// Keys returns all keys in the hash map
func (m *FlapHashMap) Keys() []uint64 {
	keys := make([]uint64, 0, m.count)

	for i := range m.buckets {
		bucket := &m.buckets[i]
		if bucket.occupied {
			keys = append(keys, bucket.key)
		}

		current := bucket.next
		for current != nil {
			keys = append(keys, current.key)
			current = current.next
		}
	}

	return keys
}

// Values returns all values in the hash map
func (m *FlapHashMap) Values() []float64 {
	values := make([]float64, 0, m.count)

	for i := range m.buckets {
		bucket := &m.buckets[i]
		if bucket.occupied {
			values = append(values, bucket.value)
		}

		current := bucket.next
		for current != nil {
			values = append(values, current.value)
			current = current.next
		}
	}

	return values
}

// Count returns the number of entries in the hash map
func (m *FlapHashMap) Count() int {
	return m.count
}

// String returns a string representation of the hash map
func (m *FlapHashMap) String() string {
	return fmt.Sprintf("FlapHashMap{count: %d, size: %d}", m.count, m.size)
}
