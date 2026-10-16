//go:build unit

package service

import (
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type delayedStreamChunkReader struct {
	chunks int
	delay  time.Duration
}

func (r *delayedStreamChunkReader) Read(p []byte) (int, error) {
	if r.chunks == 0 {
		return 0, io.EOF
	}
	time.Sleep(r.delay)
	r.chunks--
	p[0] = 'x'
	return 1, nil
}

func TestStreamActivityReaderTracksSubLineReadsPastIdleThreshold(t *testing.T) {
	var lastReadAt int64
	reader := newStreamActivityReader(&delayedStreamChunkReader{chunks: 12, delay: 200 * time.Millisecond}, &lastReadAt)
	previous := atomic.LoadInt64(&lastReadAt)
	buf := make([]byte, 1)
	started := time.Now()

	for i := 0; i < 12; i++ {
		n, err := reader.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 1, n)
		current := atomic.LoadInt64(&lastReadAt)
		require.Greater(t, current, previous, "每次读到字节都应刷新活动时间")
		require.Less(t, time.Since(time.Unix(0, current)), time.Second, "持续到达的分块不应触发 1 秒 idle timeout")
		previous = current
	}

	require.Greater(t, time.Since(started), 2*time.Second, "测试应覆盖超过 idle 阈值的持续分块")
}

func TestStreamActivityReaderDoesNotRefreshOnEmptyRead(t *testing.T) {
	var lastReadAt int64
	reader := newStreamActivityReader(emptyStreamReadError{}, &lastReadAt)
	initial := atomic.LoadInt64(&lastReadAt)

	n, err := reader.Read(make([]byte, 1))
	require.Zero(t, n)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.Equal(t, initial, atomic.LoadInt64(&lastReadAt))
}

type emptyStreamReadError struct{}

func (emptyStreamReadError) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
