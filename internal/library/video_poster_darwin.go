//go:build darwin && cgo

package library

/*
#cgo LDFLAGS: -framework Foundation -framework AVFoundation -framework CoreMedia -framework CoreGraphics -framework ImageIO
#include <stdlib.h>

void *promptVideoPoster(const char *path, int *length);
*/
import "C"

import (
	"errors"
	"unsafe"
)

func decodeVideoPoster(path string) ([]byte, error) {
	name := C.CString(path)
	defer C.free(unsafe.Pointer(name))
	var length C.int
	data := C.promptVideoPoster(name, &length)
	if data == nil {
		return nil, errors.New("无法生成视频缩略图")
	}
	defer C.free(data)
	return C.GoBytes(data, length), nil
}
