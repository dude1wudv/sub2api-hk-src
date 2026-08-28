package service

import (
	"io"
	"time"
)

type openAIRawStreamScanEvent struct {
	line string
	err  error
	done bool
}

type openAIRawStreamActivityReader struct {
	reader   io.Reader
	activity chan<- struct{}
}

func (r *openAIRawStreamActivityReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		select {
		case r.activity <- struct{}{}:
		default:
		}
	}
	return n, err
}

func (s *OpenAIGatewayService) rawChatCompletionsStreamIdleTimeout() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.Gateway.StreamDataIntervalTimeout <= 0 {
		return 0
	}
	return time.Duration(s.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
}

// scanRawChatCompletionsSSE scans the upstream stream while enforcing an idle
// timeout on upstream reads. Any bytes read from the upstream body reset the
// timer, even when they do not complete an SSE line yet.
func (s *OpenAIGatewayService) scanRawChatCompletionsSSE(
	body io.ReadCloser,
	idleTimeout time.Duration,
	handleLine func(string),
) error {
	if idleTimeout <= 0 {
		scanner := s.newUpstreamSSEScanner(body)
		for scanner.Scan() {
			handleLine(scanner.Text())
		}
		return scanner.Err()
	}

	activity := make(chan struct{}, 1)
	events := make(chan openAIRawStreamScanEvent, 1)
	stop := make(chan struct{})
	defer close(stop)

	scanner := s.newUpstreamSSEScanner(&openAIRawStreamActivityReader{
		reader:   body,
		activity: activity,
	})
	go func() {
		send := func(event openAIRawStreamScanEvent) bool {
			select {
			case events <- event:
				return true
			case <-stop:
				return false
			}
		}
		for scanner.Scan() {
			if !send(openAIRawStreamScanEvent{line: scanner.Text()}) {
				return
			}
		}
		_ = send(openAIRawStreamScanEvent{err: scanner.Err(), done: true})
	}()

	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()
	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(idleTimeout)
	}
	drainActivity := func() {
		for {
			select {
			case <-activity:
			default:
				return
			}
		}
	}
	consumeEvent := func(event openAIRawStreamScanEvent) (done bool, err error) {
		if event.done {
			return true, event.err
		}
		drainActivity()
		resetTimer()
		handleLine(event.line)
		return false, nil
	}

	for {
		select {
		case <-activity:
			resetTimer()
		case event := <-events:
			if done, err := consumeEvent(event); done {
				return err
			}
		case <-timer.C:
			// Prefer already-observed upstream progress or scanner completion over
			// a simultaneously ready timeout tick.
			select {
			case <-activity:
				resetTimer()
				continue
			default:
			}
			select {
			case event := <-events:
				if done, err := consumeEvent(event); done {
					return err
				}
				continue
			default:
			}

			_ = body.Close()
			return ErrOpenAIUpstreamStreamIdleTimeout
		}
	}
}
