// Channel runtime implementation for Flap
// Provides Go-style CSP channels with thread-safe send/receive operations
// Zero-runtime design: no GC, uses futex for efficient blocking

#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <errno.h>

// Channel states
#define CHANNEL_OPEN 0
#define CHANNEL_CLOSED 1

// Channel structure
// Layout: [mutex][cond_send][cond_recv][buffer][read_idx][write_idx][count][capacity][closed]
typedef struct {
    pthread_mutex_t mutex;
    pthread_cond_t cond_send;   // Signal when space available
    pthread_cond_t cond_recv;   // Signal when data available
    double* buffer;              // Ring buffer for values
    size_t read_idx;
    size_t write_idx;
    size_t count;                // Current number of items
    size_t capacity;             // Buffer capacity (0 = unbuffered)
    int closed;
} Channel;

// Create a new channel
// capacity: 0 for unbuffered, >0 for buffered
void* channel_create(size_t capacity) {
    Channel* ch = (Channel*)malloc(sizeof(Channel));
    if (!ch) return NULL;

    pthread_mutex_init(&ch->mutex, NULL);
    pthread_cond_init(&ch->cond_send, NULL);
    pthread_cond_init(&ch->cond_recv, NULL);

    ch->buffer = capacity > 0 ? (double*)malloc(capacity * sizeof(double)) : NULL;
    ch->read_idx = 0;
    ch->write_idx = 0;
    ch->count = 0;
    ch->capacity = capacity;
    ch->closed = CHANNEL_OPEN;

    return (void*)ch;
}

// Send a value to the channel (blocking)
// Returns: 0 on success, -1 if channel is closed
int channel_send(void* channel, double value) {
    if (!channel) return -1;

    Channel* ch = (Channel*)channel;
    pthread_mutex_lock(&ch->mutex);

    // Check if closed
    if (ch->closed) {
        pthread_mutex_unlock(&ch->mutex);
        return -1;  // Cannot send to closed channel
    }

    // For buffered channels: wait until space available
    if (ch->capacity > 0) {
        while (ch->count >= ch->capacity && !ch->closed) {
            pthread_cond_wait(&ch->cond_send, &ch->mutex);
        }

        if (ch->closed) {
            pthread_mutex_unlock(&ch->mutex);
            return -1;
        }

        // Add to buffer
        ch->buffer[ch->write_idx] = value;
        ch->write_idx = (ch->write_idx + 1) % ch->capacity;
        ch->count++;

        // Signal receiver
        pthread_cond_signal(&ch->cond_recv);
    } else {
        // Unbuffered: direct handoff
        // For now, store value temporarily and wait for receiver
        // This is a simplified implementation
        if (ch->count > 0) {
            // Already has a value waiting, block until consumed
            while (ch->count > 0 && !ch->closed) {
                pthread_cond_wait(&ch->cond_send, &ch->mutex);
            }
        }

        if (ch->closed) {
            pthread_mutex_unlock(&ch->mutex);
            return -1;
        }

        // Use buffer[0] for unbuffered handoff
        if (!ch->buffer) {
            ch->buffer = (double*)malloc(sizeof(double));
        }
        ch->buffer[0] = value;
        ch->count = 1;

        // Signal receiver
        pthread_cond_signal(&ch->cond_recv);

        // Wait for receiver to consume
        while (ch->count > 0 && !ch->closed) {
            pthread_cond_wait(&ch->cond_send, &ch->mutex);
        }
    }

    pthread_mutex_unlock(&ch->mutex);
    return 0;
}

// Receive a value from the channel (blocking)
// Returns: value from channel, or 0.0 if channel is closed and empty
double channel_recv(void* channel) {
    if (!channel) return 0.0;

    Channel* ch = (Channel*)channel;
    pthread_mutex_lock(&ch->mutex);

    // Wait until data available or channel closed
    while (ch->count == 0 && !ch->closed) {
        pthread_cond_wait(&ch->cond_recv, &ch->mutex);
    }

    // If closed and empty, return 0
    if (ch->count == 0 && ch->closed) {
        pthread_mutex_unlock(&ch->mutex);
        return 0.0;
    }

    // Get value from buffer
    double value;
    if (ch->capacity > 0) {
        value = ch->buffer[ch->read_idx];
        ch->read_idx = (ch->read_idx + 1) % ch->capacity;
        ch->count--;

        // Signal sender that space is available
        pthread_cond_signal(&ch->cond_send);
    } else {
        // Unbuffered: get value and signal sender
        value = ch->buffer[0];
        ch->count = 0;

        // Signal sender that value was consumed
        pthread_cond_signal(&ch->cond_send);
    }

    pthread_mutex_unlock(&ch->mutex);
    return value;
}

// Close the channel
// After closing, sends will fail and receives will return 0 when empty
void channel_close(void* channel) {
    if (!channel) return;

    Channel* ch = (Channel*)channel;
    pthread_mutex_lock(&ch->mutex);

    ch->closed = CHANNEL_CLOSED;

    // Wake up all blocked senders and receivers
    pthread_cond_broadcast(&ch->cond_send);
    pthread_cond_broadcast(&ch->cond_recv);

    pthread_mutex_unlock(&ch->mutex);
}

// Destroy the channel and free resources
void channel_destroy(void* channel) {
    if (!channel) return;

    Channel* ch = (Channel*)channel;

    pthread_mutex_destroy(&ch->mutex);
    pthread_cond_destroy(&ch->cond_send);
    pthread_cond_destroy(&ch->cond_recv);

    if (ch->buffer) {
        free(ch->buffer);
    }

    free(ch);
}

// Check if channel is closed
int channel_is_closed(void* channel) {
    if (!channel) return 1;

    Channel* ch = (Channel*)channel;
    pthread_mutex_lock(&ch->mutex);
    int closed = ch->closed;
    pthread_mutex_unlock(&ch->mutex);

    return closed;
}
