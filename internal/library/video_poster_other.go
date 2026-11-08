//go:build !darwin || !cgo

package library

import "errors"

func decodeVideoPoster(string) ([]byte, error) {
	return nil, errors.ErrUnsupported
}
