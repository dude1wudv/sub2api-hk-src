package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

type chatSpeechSocket interface {
	Read(context.Context) (coderws.MessageType, []byte, error)
	Write(context.Context, coderws.MessageType, []byte) error
	CloseNow() error
}

type chatSpeechDial func(context.Context, string, http.Header, string) (chatSpeechSocket, error)

func dialChatSpeech(ctx context.Context, target string, headers http.Header, proxy string) (chatSpeechSocket, error) {
	shared, err := httpclient.GetClient(httpclient.Options{ProxyURL: proxy, ValidateResolvedIP: true, ResponseHeaderTimeout: 30 * time.Second})
	if err != nil {
		return nil, err
	}
	client := *shared
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	conn, response, err := coderws.Dial(ctx, target, &coderws.DialOptions{HTTPClient: &client, HTTPHeader: headers, CompressionMode: coderws.CompressionDisabled})
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, errors.New("speech upstream connection failed")
	}
	conn.SetReadLimit(2 << 20)
	return conn, nil
}

func (s *OpenAIGatewayService) chatSpeechAccount(ctx context.Context, key *APIKey, account *Account, kind string) (*Account, error) {
	id, err := s.ValidateChatAuxiliaryScope(ctx, key, kind)
	if err != nil || account == nil || account.ID != id {
		return nil, ErrChatAuxiliaryUnavailable
	}
	current, err := s.SelectChatAuxiliaryAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err = s.validateUpstreamBaseURL("https://api.deepseek.com"); err != nil {
		return nil, ErrChatAuxiliaryUnavailable
	}
	return current, nil
}

func chatSpeechResult(ctx context.Context, kind string, started time.Time, quantity *int64, status string) *OpenAIForwardResult {
	usage, _ := chatAuxiliaryUsage(kind, quantity, status)
	scope := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	return &OpenAIForwardResult{RequestID: "chat_auxiliary:" + scope.operationID.String(), Model: "deepseek-" + kind, UpstreamEndpoint: "/v1/tools/" + kind, Duration: time.Since(started), chatAuxiliaryOnly: true, ChatAuxiliaryUsage: usage}
}

func (s *OpenAIGatewayService) ForwardChatASR(ctx context.Context, key *APIKey, account *Account, input *ChatAuxiliaryInput) (result *OpenAIForwardResult, body []byte, err error) {
	account, err = s.chatSpeechAccount(ctx, key, account, "asr")
	if err != nil {
		return nil, nil, err
	}
	request, err := http.NewRequestWithContext(WithHTTPUpstreamRedirectsDisabled(ctx), http.MethodPost, "https://api.deepseek.com/v1/tools/asr", bytes.NewReader(input.Body))
	if err != nil {
		return nil, nil, err
	}
	request.Header.Set("Authorization", "Bearer "+account.GetCredential("api_key"))
	request.Header.Set("Content-Type", input.ContentType)
	request.Header.Set("Accept", "application/json")
	started := time.Now()
	response, err := s.httpUpstream.Do(request, resolveAccountProxyURL(account), account.ID, account.Concurrency)
	if err != nil {
		return nil, nil, err
	}
	if response == nil || response.Body == nil {
		return nil, nil, ErrChatStream
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, nil, ErrChatStream
	}
	result = chatSpeechResult(ctx, "asr", started, nil, "incomplete")
	body, err = readChatBounded(response.Body, 256<<10)
	if err != nil {
		return result, nil, err
	}
	var payload struct {
		RequestID  string `json:"request_id"`
		Text       string `json:"text"`
		DurationMS *int64 `json:"duration_ms"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.DurationMS == nil || payload.RequestID == "" {
		return result, nil, ErrChatStream
	}
	quantity, err := chatAuxiliaryASRSeconds(*payload.DurationMS)
	if err != nil {
		return result, nil, ErrChatStream
	}
	result = chatSpeechResult(ctx, "asr", started, &quantity, "confirmed")
	body, err = json.Marshal(payload)
	return result, body, err
}

func chatSpeechEvent(w http.ResponseWriter, name string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, raw); err != nil {
		return err
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

// A normal HTTP EOF cannot prove streamed TTS finished. Use the provider's
// explicit WebSocket success event, then attest every forwarded PCM byte.
func (s *OpenAIGatewayService) ForwardChatTTS(ctx context.Context, c *gin.Context, key *APIKey, account *Account, input *ChatAuxiliaryInput) (result *OpenAIForwardResult, err error) {
	account, err = s.chatSpeechAccount(ctx, key, account, "tts")
	if err != nil {
		return nil, err
	}
	quantity, err := ValidateChatTTS(input.TTS)
	if err != nil {
		return nil, err
	}
	query := url.Values{"format": {"pcm"}}
	if input.TTS.VoiceID != "" {
		query.Set("voice_id", input.TTS.VoiceID)
	}
	if input.TTS.FilterMarkdown != nil {
		query.Set("filter_markdown", fmt.Sprint(*input.TTS.FilterMarkdown))
	}
	headers := http.Header{"Authorization": {"Bearer " + account.GetCredential("api_key")}}
	dial := s.chatTTSDial
	if dial == nil {
		dial = dialChatSpeech
	}
	started := time.Now()
	conn, err := dial(ctx, "wss://api.deepseek.com/v1/tools/tts?"+query.Encode(), headers, resolveAccountProxyURL(account))
	if err != nil {
		return nil, err
	}
	defer conn.CloseNow()
	result = chatSpeechResult(ctx, "tts", started, &quantity, "incomplete")
	result.Stream = true
	defer func() {
		result.Duration = time.Since(started)
		if err != nil && c.Writer.Written() && ctx.Err() == nil {
			_ = chatSpeechEvent(c.Writer, "error", map[string]any{"code": "CHAT_TTS_INCOMPLETE", "message": "语音传输未完整结束，请勿将部分音频视为完整结果", "metering_status": result.ChatAuxiliaryUsage.Items[0].MeteringStatus})
		}
	}()
	text, _ := json.Marshal(map[string]string{"event": "text", "text": input.TTS.Text})
	if err = conn.Write(ctx, coderws.MessageText, text); err != nil {
		return result, err
	}
	if err = conn.Write(ctx, coderws.MessageText, []byte(`{"event":"finish_text"}`)); err != nil {
		return result, err
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Accel-Buffering", "no")
	if err = chatSpeechEvent(c.Writer, "audio_start", map[string]any{"format": "pcm", "sample_rate": 24000, "channels": 1, "bits_per_sample": 16, "request_id": result.RequestID}); err != nil {
		return result, err
	}
	digest := sha256.New()
	total, sequence := int64(0), 0
	for {
		var kind coderws.MessageType
		var data []byte
		kind, data, err = conn.Read(ctx)
		if err != nil {
			return result, ErrChatStream
		}
		switch kind {
		case coderws.MessageBinary:
			total += int64(len(data))
			if total > 24000*2*30*60 {
				return result, ErrChatTooLarge
			}
			_, _ = digest.Write(data)
			if err = chatSpeechEvent(c.Writer, "audio", map[string]any{"sequence": sequence, "data": base64.StdEncoding.EncodeToString(data)}); err != nil {
				return result, err
			}
			sequence++
		case coderws.MessageText:
			var finish struct {
				Event string `json:"event"`
				Code  string `json:"code"`
			}
			if json.Unmarshal(data, &finish) != nil || finish.Event != "finish" || finish.Code != "success" || total == 0 || total%2 != 0 {
				return result, ErrChatStream
			}
			// Bill only an authoritative success. Downstream cancellation after this
			// point must not erase the provider's completed operation.
			result.ChatAuxiliaryUsage, _ = chatAuxiliaryUsage("tts", &quantity, "confirmed")
			err = chatSpeechEvent(c.Writer, "audio_complete", map[string]any{"bytes": total, "chunks": sequence, "sha256": hex.EncodeToString(digest.Sum(nil)), "request_id": result.RequestID})
			return result, err
		default:
			return result, ErrChatStream
		}
	}
}
