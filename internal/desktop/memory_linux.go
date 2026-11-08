package desktop

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func currentMemoryUsage() MemoryUsage {
	var result MemoryUsage
	var info syscall.Sysinfo_t
	if syscall.Sysinfo(&info) == nil {
		result.TotalBytes = uint64(info.Totalram) * uint64(info.Unit)
	}
	raw, err := os.ReadFile("/proc/self/statm")
	if err == nil {
		fields := strings.Fields(string(raw))
		if len(fields) > 1 {
			pages, err := strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				result.AppBytes = pages * uint64(os.Getpagesize())
			}
		}
	}
	return result
}
