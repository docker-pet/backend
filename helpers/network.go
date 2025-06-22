package helpers

import (
	"errors"
	"io"
	"github.com/Jeffail/gabs/v2"
)

const maxBodySize = 15 * 1024 // 15 KB

var ErrBodyTooLarge = errors.New("request body exceeds 15KB limit")

func ParseJSONBodyLimited(r io.ReadCloser) (*gabs.Container, error) {
	defer r.Close()

	limited := io.LimitReader(r, maxBodySize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	if len(data) > maxBodySize {
		return nil, ErrBodyTooLarge
	}

	return gabs.ParseJSON(data)
}