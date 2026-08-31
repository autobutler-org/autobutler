package vfs

import "io"

// readBounded reads r whole, refusing more than [MaxInMemoryWriteBytes]. It
// reads one byte past the cap so an oversized write is rejected rather than
// silently truncated into a half-stored file.
func readBounded(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxInMemoryWriteBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxInMemoryWriteBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}
