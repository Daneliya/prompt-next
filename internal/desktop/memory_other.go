//go:build (!darwin && !windows && !linux) || (darwin && !cgo)

package desktop

func currentMemoryUsage() MemoryUsage {
	return MemoryUsage{}
}
