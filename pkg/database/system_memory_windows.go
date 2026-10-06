//go:build windows
// +build windows

package database

import (
	"syscall"
	"unsafe"
)

type memoryStatusEx struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx  = kernel32.NewProc("GlobalMemoryStatusEx")
)

func systemMemoryBytes() int64 {
	var status memoryStatusEx
	status.cbSize = uint32(unsafe.Sizeof(status))
	r1, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if r1 == 0 {
		return 0
	}
	return int64(status.ullTotalPhys)
}
