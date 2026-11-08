package desktop

import (
	"syscall"
	"unsafe"
)

var (
	memoryKernel32 = syscall.NewLazyDLL("kernel32.dll")
	memoryStatus   = memoryKernel32.NewProc("GlobalMemoryStatusEx")
	processMemory  = memoryKernel32.NewProc("K32GetProcessMemoryInfo")
)

func currentMemoryUsage() MemoryUsage {
	status := struct {
		length, load uint32
		totalPhys    uint64
		other        [6]uint64
	}{}
	status.length = uint32(unsafe.Sizeof(status))
	var result MemoryUsage
	if ok, _, _ := memoryStatus.Call(uintptr(unsafe.Pointer(&status))); ok != 0 {
		result.TotalBytes = status.totalPhys
	}
	counters := struct {
		length, pageFaults         uint32
		peakWorkingSet, workingSet uintptr
		other                      [6]uintptr
	}{}
	counters.length = uint32(unsafe.Sizeof(counters))
	currentProcess := ^uintptr(0) // Windows pseudo-handle for the calling process.
	if ok, _, _ := processMemory.Call(currentProcess, uintptr(unsafe.Pointer(&counters)), uintptr(counters.length)); ok != 0 {
		result.AppBytes = uint64(counters.workingSet)
	}
	return result
}
