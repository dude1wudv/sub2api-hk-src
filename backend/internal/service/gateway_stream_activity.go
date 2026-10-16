package service

import (
	"io"
	"sync/atomic"
	"time"
)

// streamActivityReader tracks upstream byte activity independently of SSE line
// boundaries. A large event can arrive in many reads before Scanner emits a line.
type streamActivityReader struct {
	reader     io.Reader
	lastReadAt *int64
}

func newStreamActivityReader(reader io.Reader, lastReadAt *int64) io.Reader {
	atomic.StoreInt64(lastReadAt, time.Now().UnixNano())
	return &streamActivityReader{reader: reader, lastReadAt: lastReadAt}
}

func (r *streamActivityReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		atomic.StoreInt64(r.lastReadAt, time.Now().UnixNano())
	}
	return n, err
}
