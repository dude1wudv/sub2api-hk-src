package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func deepSeekTTSQuery(query url.Values) (url.Values, error) {
	out := url.Values{}
	for key, values := range query {
		if len(values) != 1 {
			return nil, ErrChatInvalid
		}
		value := values[0]
		switch key {
		case "voice_id":
			if value != "stella" && value != "tide" {
				return nil, ErrChatInvalid
			}
		case "format":
			if value != "pcm" && value != "opus" {
				return nil, ErrChatInvalid
			}
		case "filter_markdown":
			if value != "true" && value != "false" {
				return nil, ErrChatInvalid
			}
		default:
			return nil, ErrChatInvalid
		}
		out.Set(key, value)
	}
	return out, nil
}

func (s *OpenAIGatewayService) ForwardDeepSeekTTSWebSocket(ctx context.Context, c *gin.Context, key *APIKey, account *Account, op *DeepSeekSuiteOperation, audit func(string) bool) (result *OpenAIForwardResult, err error) {
	defer func() {
		if result == nil {
			op.Status = "failed"
		}
	}()
	query, err := deepSeekTTSQuery(c.Request.URL.Query())
	if err != nil {
		return nil, err
	}
	account, err = s.DeepSeekSuiteAccount(ctx, key, account, "tts_ws")
	if err != nil {
		return nil, err
	}
	target, err := s.deepSeekSuiteURL(account, "tts_ws", "")
	if err != nil {
		return nil, err
	}
	u, _ := url.Parse(target)
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	q := u.Query()
	for k, values := range query {
		q[k] = values
	}
	u.RawQuery = q.Encode()
	dial := s.chatTTSDial
	if dial == nil {
		dial = dialChatSpeech
	}
	// Only the selected provider credential reaches upstream. Neither the
	// gateway bearer nor the client's credential-bearing subprotocol is copied.
	upstream, err := dial(ctx, u.String(), http.Header{"Authorization": {"Bearer " + account.GetCredential("api_key")}}, resolveAccountProxyURL(account))
	if err != nil {
		return nil, err
	}
	defer func() { _ = upstream.CloseNow() }()
	client, err := coderws.Accept(c.Writer, c.Request, &coderws.AcceptOptions{
		Subprotocols: []string{"tts"}, CompressionMode: coderws.CompressionDisabled,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.CloseNow() }()
	client.SetReadLimit(64 << 10)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	started := time.Now()
	result = suiteResult(op, &DeepSeekSuiteInput{Kind: "tts", Stream: true})
	result.UpstreamEndpoint = u.Path
	var chars atomic.Int64
	var finishedText atomic.Bool
	type event struct {
		kind coderws.MessageType
		data []byte
		err  error
	}
	events := make(chan event, 2)
	emit := func(value event) {
		select {
		case events <- value:
		case <-ctx.Done():
		}
	}
	go func() {
		var totalBytes int
		for {
			read, stop := context.WithTimeout(ctx, 60*time.Second)
			kind, raw, readErr := client.Read(read)
			stop()
			if readErr != nil {
				emit(event{err: readErr})
				return
			}
			totalBytes += len(raw)
			if kind != coderws.MessageText || totalBytes > 64<<10 || !validDeepSeekSuiteJSON(raw) || finishedText.Load() {
				emit(event{err: ErrChatInvalid})
				return
			}
			switch gjson.GetBytes(raw, "event").String() {
			case "text":
				value := gjson.GetBytes(raw, "text")
				count, countErr := ChatAuxiliaryTTSCharacters(value.String())
				if value.Type != gjson.String || countErr != nil || chars.Load()+count > 10000 || (audit != nil && !audit(value.String())) {
					emit(event{err: ErrChatInvalid})
					return
				}
				chars.Add(count)
			case "finish_text":
				finishedText.Store(true)
			default:
				emit(event{err: ErrChatInvalid})
				return
			}
			if writeErr := upstream.Write(ctx, coderws.MessageText, raw); writeErr != nil {
				emit(event{err: writeErr})
				return
			}
		}
	}()
	go func() {
		for {
			read, stop := context.WithTimeout(ctx, 60*time.Second)
			kind, data, readErr := upstream.Read(read)
			stop()
			emit(event{kind: kind, data: data, err: readErr})
			if readErr != nil || ctx.Err() != nil {
				return
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ping, stop := context.WithTimeout(ctx, 5*time.Second)
				pingErr := client.Ping(ping)
				if remote, ok := upstream.(interface{ Ping(context.Context) error }); ok && pingErr == nil {
					pingErr = remote.Ping(ping)
				}
				stop()
				if pingErr != nil {
					emit(event{err: pingErr})
					return
				}
			}
		}
	}()
	status := "incomplete"
	defer func() {
		quantity := chars.Load()
		result.ChatAuxiliaryUsage, _ = chatAuxiliaryUsage("tts", &quantity, status)
		result.Duration = time.Since(started)
	}()
	var audioBytes int64
	for {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case incoming := <-events:
			if incoming.err != nil {
				_ = client.Close(coderws.StatusPolicyViolation, "Incomplete DeepSeek speech session")
				return result, incoming.err
			}
			if incoming.kind == coderws.MessageBinary {
				audioBytes += int64(len(incoming.data))
				if audioBytes > 24000*2*30*60 {
					return result, ErrChatTooLarge
				}
			}
			if incoming.kind == coderws.MessageText && gjson.GetBytes(incoming.data, "event").String() == "finish" {
				if gjson.GetBytes(incoming.data, "code").String() == "success" && finishedText.Load() {
					// Provider completion is billable even if the last client
					// frame cannot be delivered.
					status = "confirmed"
				}
				if writeErr := client.Write(ctx, incoming.kind, incoming.data); writeErr != nil {
					return result, writeErr
				}
				_ = client.Close(coderws.StatusNormalClosure, "finish")
				if status != "confirmed" {
					return result, errors.New("upstream speech did not complete")
				}
				return result, nil
			}
			if writeErr := client.Write(ctx, incoming.kind, incoming.data); writeErr != nil {
				return result, writeErr
			}
		}
	}
}
