//go:build darwin && cgo

package desktop

/*
#include <stdint.h>
#include <sys/sysctl.h>
#include <mach/mach.h>

static uint64_t physicalMemory(void) {
	uint64_t bytes = 0;
	size_t length = sizeof(bytes);
	if (sysctlbyname("hw.memsize", &bytes, &length, NULL, 0) != 0) {
		return 0;
	}
	return bytes;
}

static uint64_t processMemory(void) {
	task_vm_info_data_t info;
	mach_msg_type_number_t count = TASK_VM_INFO_COUNT;
	if (task_info(mach_task_self(), TASK_VM_INFO, (task_info_t)&info, &count) != KERN_SUCCESS ||
		count < TASK_VM_INFO_REV1_COUNT) {
		return 0;
	}
	return info.phys_footprint;
}
*/
import "C"

func currentMemoryUsage() MemoryUsage {
	return MemoryUsage{AppBytes: uint64(C.processMemory()), TotalBytes: uint64(C.physicalMemory())}
}
