//go:build !linux
// +build !linux

package main

import (
	"fmt"
	"time"
)

type CodePage struct {
	addr      uintptr
	size      int
	code      []byte
	allocated time.Time
}

type HotReloadManager struct {
	activePages map[string]*CodePage
	oldPages    []*CodePage
	gracePeriod time.Duration
}

func NewHotReloadManager() *HotReloadManager {
	return &HotReloadManager{
		activePages: make(map[string]*CodePage),
		oldPages:    make([]*CodePage, 0),
		gracePeriod: 1 * time.Second,
	}
}

func (hrm *HotReloadManager) AllocateExecutablePage(size int) (*CodePage, error) {
	return nil, fmt.Errorf("hot reloading not supported on this platform")
}

func (page *CodePage) CopyCode(code []byte) error {
	return fmt.Errorf("hot reloading not supported on this platform")
}

func (page *CodePage) GetAddress() uintptr {
	return 0
}

func (hrm *HotReloadManager) LoadHotFunction(name string, code []byte) (uintptr, error) {
	return 0, fmt.Errorf("hot reloading not supported on this platform")
}

func (hrm *HotReloadManager) cleanupOldPages() {
}

func (hrm *HotReloadManager) FreePage(page *CodePage) error {
	return nil
}

func UpdateFunctionPointer(tableAddr uintptr, index int, newAddr uintptr) {
}

func ExtractFunctionCode(elfPath string, functionName string) ([]byte, error) {
	return nil, fmt.Errorf("ELF extraction not supported on this platform")
}

func (hrm *HotReloadManager) ReloadHotFunction(name string, code []byte, tableAddr uintptr, tableIndex int) error {
	return fmt.Errorf("hot reloading not supported on this platform")
}
