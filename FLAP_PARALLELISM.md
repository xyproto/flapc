# Flap CPU Parallelism

## Philosophy

**Keep it minimal**: A simple numeric prefix on loops enables parallel execution.

**No GPU complexity**: Use Flap's C FFI to call Vulkan/CUDA/Metal libraries directly.

---

## Parallel Loop Syntax

```flap
// Sequential loop (default)
@ item in collection max 10000 {
    process(item)
}

// Parallel loop with N cores
N @ item in collection max 10000 {
    process(item)
}
```

The number before `@` specifies **how many CPU cores/threads** to use for parallel execution.

---

## Examples

### Basic Parallelism

```flap
import parallel

main ==> {
    data := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    results := []

    // Sequential processing
    @ x in data max 100 {
        results <- results + [x ** 2]
    }

    // Parallel processing with 4 cores
    parallel_results := []
    4 @ x in data max 100 {
        parallel_results <- parallel_results + [x ** 2]
    }
}
```

### Image Processing

```flap
// Process image in parallel across all cores
process_image := (image) => {
    width := image.width
    height := image.height
    cores := cpu_count()

    output := create_image(width, height)

    // Split work across all CPU cores
    cores @ y in range(0, height) max 10000 {
        @ x in range(0, width) max 10000 {
            pixel := image[y * width + x]
            output[y * width + x] <- blur(pixel, image, x, y)
        }
    }

    -> output
}
```

### Ray Tracing

```flap
render_scene := (scene, width, height) => {
    cores := cpu_count()
    pixels := []

    // Render pixels in parallel
    cores @ y in range(0, height) max 10000 {
        @ x in range(0, width) max 10000 {
            ray := generate_ray(x, y, width, height)
            color := trace_ray(ray, scene, depth: 5)
            pixels <- pixels + [color]
        }
    }

    -> pixels
}
```

### Particle Physics

```flap
// Update particles in parallel
update_particles := (particles, dt) => {
    cores := cpu_count()

    cores @ particle in particles max 100000 {
        // Update position
        particle.x <- particle.x + particle.vx * dt
        particle.y <- particle.y + particle.vy * dt
        particle.z <- particle.z + particle.vz * dt

        // Apply gravity
        particle.vy <- particle.vy + gravity * dt

        // Collision detection (each thread handles subset)
        check_collisions(particle, particles)
    }
}
```

### Matrix Multiplication

```flap
// Parallel matrix multiplication
matmul := (A, B, n) => {
    C := create_matrix(n, n)
    cores := cpu_count()

    // Each core handles subset of rows
    cores @ i in range(0, n) max 10000 {
        @ j in range(0, n) max 10000 {
            sum := 0.0
            @ k in range(0, n) max 10000 {
                sum <- sum + A[i * n + k] * B[k * n + j]
            }
            C[i * n + j] <- sum
        }
    }

    -> C
}
```

### Data Processing Pipeline

```flap
// Process large dataset in parallel
process_dataset := (data) => {
    cores := cpu_count()

    // Stage 1: Parse in parallel
    parsed := []
    cores @ item in data max 1000000 {
        parsed <- parsed + [parse(item)]
    }

    // Stage 2: Transform in parallel
    transformed := []
    cores @ item in parsed max 1000000 {
        transformed <- transformed + [transform(item)]
    }

    // Stage 3: Aggregate (sequential, needs ordering)
    result := 0
    @ item in transformed max 1000000 {
        result <- result + item.value
    }

    -> result
}
```

---

## Controlling Parallelism

```flap
// Use all available cores
cores := cpu_count()
cores @ item in data max 100000 {
    process(item)
}

// Use half the cores (for thermal/power reasons)
half_cores := cpu_count() / 2
half_cores @ item in data max 100000 {
    process(item)
}

// Fixed number of cores
4 @ item in data max 100000 {
    process(item)
}

// Adaptive parallelism based on data size
parallel_count := data.length > 1000 { cpu_count() ~> 1 }
parallel_count @ item in data max 100000 {
    process(item)
}
```

---

## Thread Safety Considerations

### Atomic Operations

```flap
// Counter needs atomic increment
counter := atomic(0)

cores @ item in data max 100000 {
    process(item)
    atomic_add(counter, 1)
}

printf("Processed %v items\n", atomic_load(counter))
```

### Mutex for Shared State

```flap
// Protect shared data structure
results := []
mutex := mutex_create()

4 @ item in data max 100000 {
    result := expensive_computation(item)

    mutex_lock(mutex)
    results <- results + [result]
    mutex_unlock(mutex)
}
```

### Thread-Local Storage

```flap
// Each thread accumulates locally, then merge
4 @ item in data max 100000 {
    thread_local := thread_local_get("accumulator", default: [])
    thread_local <- thread_local + [process(item)]
    thread_local_set("accumulator", thread_local)
}

// Merge thread-local results
all_results := []
@ i in range(0, cpu_count()) max 100 {
    thread_result := thread_get_result(i, "accumulator")
    all_results <- all_results + thread_result
}
```

---

## Implementation Strategy

### Runtime Thread Pool

```c
// Flap runtime provides thread pool
typedef struct {
    pthread_t *threads;
    int num_threads;
    work_queue_t *queue;
} flap_thread_pool_t;

flap_thread_pool_t* flap_thread_pool_create(int num_threads);
void flap_thread_pool_execute(flap_thread_pool_t *pool, void (*func)(void*), void *arg);
void flap_thread_pool_wait(flap_thread_pool_t *pool);
```

### Parallel Loop Compilation

```flap
// Source code
4 @ x in data max 1000 {
    process(x)
}

// Compiles to (conceptually):
pool := _flap_get_thread_pool(4)
chunk_size := data.length / 4

@ thread_id in range(0, 4) max 100 {
    start := thread_id * chunk_size
    end := start + chunk_size

    _flap_thread_pool_execute(pool, (tid) => {
        @ i in range(start, end) max 1000 {
            x := data[i]
            process(x)
        }
    })
}

_flap_thread_pool_wait(pool)
```

---

## GPU Compute via C FFI

For GPU workloads, use Flap's C FFI to call GPU libraries directly:

### Vulkan Compute

```flap
// Call Vulkan compute shader via C FFI
vulkan_compute := (data, size) => {
    // C function: void* vulkan_blur(float* data, int size)
    result := call_c("vulkan_blur", [data, size])
    -> result
}
```

### CUDA

```flap
// Call CUDA kernel via C FFI
cuda_matmul := (A, B, n) => {
    // C function: void cuda_matmul(float* A, float* B, float* C, int n)
    C := allocate_buffer(n * n)
    call_c("cuda_matmul", [A, B, C, n])
    -> C
}
```

### Metal (macOS)

```flap
// Call Metal compute via C FFI
metal_compute := (data, size) => {
    result := call_c("metal_compute_shader", [data, size])
    -> result
}
```

This approach:
- Keeps Flap minimal and focused
- Leverages existing GPU ecosystems
- No need for GPU language integration
- Optimal performance through native code

---

## Summary

**CPU Parallelism:**
- Prefix loops with number of cores: `N @ item in collection`
- Optional prefix (default is sequential)
- Simple, clean, minimal

**GPU Compute:**
- Use C FFI to call Vulkan/CUDA/Metal
- No built-in GPU language complexity
- Leverage existing ecosystems

**Philosophy:**
- Keep the language small
- Provide powerful primitives
- Let libraries handle complexity
- Trust the FFI for specialization
