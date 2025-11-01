//go:build !linux
// +build !linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileWatcher struct {
	files       []string
	modTimes    map[string]time.Time
	mu          sync.Mutex
	debounceMap map[string]*time.Timer
	onChange    func(string)
	stopChan    chan struct{}
}

func NewFileWatcher(onChange func(string)) (*FileWatcher, error) {
	return &FileWatcher{
		files:       make([]string, 0),
		modTimes:    make(map[string]time.Time),
		debounceMap: make(map[string]*time.Timer),
		onChange:    onChange,
		stopChan:    make(chan struct{}),
	}, nil
}

func (fw *FileWatcher) AddFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %v", absPath, err)
	}

	fw.mu.Lock()
	fw.files = append(fw.files, absPath)
	fw.modTimes[absPath] = info.ModTime()
	fw.mu.Unlock()

	return nil
}

func (fw *FileWatcher) Watch() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fw.checkFiles()
		case <-fw.stopChan:
			return
		}
	}
}

func (fw *FileWatcher) checkFiles() {
	fw.mu.Lock()
	files := make([]string, len(fw.files))
	copy(files, fw.files)
	fw.mu.Unlock()

	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Error stating file %s: %v\n", path, err)
			}
			continue
		}

		fw.mu.Lock()
		oldModTime := fw.modTimes[path]
		fw.mu.Unlock()

		if info.ModTime().After(oldModTime) {
			fw.mu.Lock()
			fw.modTimes[path] = info.ModTime()
			fw.mu.Unlock()

			fw.debouncedCallback(path)
		}
	}
}

func (fw *FileWatcher) debouncedCallback(path string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if timer, exists := fw.debounceMap[path]; exists {
		timer.Stop()
	}

	fw.debounceMap[path] = time.AfterFunc(500*time.Millisecond, func() {
		fw.onChange(path)
		fw.mu.Lock()
		delete(fw.debounceMap, path)
		fw.mu.Unlock()
	})
}

func (fw *FileWatcher) Close() error {
	close(fw.stopChan)
	return nil
}
