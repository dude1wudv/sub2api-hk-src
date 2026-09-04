package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"
)

const (
	QwenAudioHTTPBaseURLExtraKey = "qwen_audio_http_base_url"
	QwenAudioWSBaseURLExtraKey   = "qwen_audio_ws_base_url"

	QwenAudioMaxWAVBytes              int64 = 7 << 20
	QwenAudioMaxRequestBodyBytes            = QwenAudioMaxWAVBytes + (1 << 20)
	QwenAudioMaxDuration                    = 5 * time.Minute
	QwenAudioMaxTTSCharacters               = 4_000
	QwenAudioMaxInstructionCharacters       = 4_000

	qwenASRPath             = "/api/v1/services/aigc/multimodal-generation/generation"
	qwenTTSPath             = "/api-ws/v1/inference"
	qwenASRResponseMaxBytes = 1 << 20
	qwenTTSOutputMaxBytes   = 32 << 20
	qwenAudioDialTimeout    = 12 * time.Second
	qwenAudioFirstEventWait = 15 * time.Second
	qwenASRTotalTimeout     = 2 * time.Minute
	qwenTTSTotalTimeout     = 2 * time.Minute
	// DashScope allows up to 200,000 cumulative characters in one TTS task.
	// This provider-side terminal usage guard is intentionally distinct from
	// the smaller product ingress limit above.
	qwenAudioProviderMaxTaskCharacters = 200_000
)

var (
	errQwenAudioOutputTooLarge = errors.New("qwen audio output exceeds limit")
	errQwenAudioProtocol       = errors.New("invalid qwen audio protocol response")
)

// WAVMetadata is the validated subset needed by Qwen and duration billing.
// Only uncompressed PCM is accepted because Hermes' conversion produces PCM
// WAV and compressed RIFF variants cannot be timed safely from data length.
type WAVMetadata struct {
	SampleRate    int
	Channels      int
	BitsPerSample int
	DataBytes     int64
	Duration      time.Duration
}

// ParsePCM16WAV validates a RIFF/WAVE PCM file and computes duration from the
// format byte rate and the data chunk, never from HTTP latency or file size.
func ParsePCM16WAV(data []byte) (WAVMetadata, error) {
	if int64(len(data)) > QwenAudioMaxWAVBytes {
		return WAVMetadata{}, fmt.Errorf("WAV file exceeds %d bytes", QwenAudioMaxWAVBytes)
	}
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return WAVMetadata{}, errors.New("file must be a valid little-endian RIFF/WAVE")
	}
	declared := int64(binary.LittleEndian.Uint32(data[4:8])) + 8
	if declared < 12 || declared > int64(len(data)) {
		return WAVMetadata{}, errors.New("WAV RIFF length is invalid")
	}

	var sampleRate, channels, bitsPerSample, byteRate, blockAlign int
	var dataBytes int64 = -1
	for offset := 12; offset+8 <= int(declared); {
		chunkSize := int64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkStart := int64(offset + 8)
		chunkEnd := chunkStart + chunkSize
		if chunkSize < 0 || chunkEnd > declared || chunkEnd > int64(len(data)) {
			return WAVMetadata{}, errors.New("WAV chunk length is invalid")
		}
		switch string(data[offset : offset+4]) {
		case "fmt ":
			if chunkSize < 16 {
				return WAVMetadata{}, errors.New("WAV fmt chunk is too short")
			}
			fmtData := data[chunkStart:chunkEnd]
			if binary.LittleEndian.Uint16(fmtData[0:2]) != 1 {
				return WAVMetadata{}, errors.New("WAV must use uncompressed PCM")
			}
			channels = int(binary.LittleEndian.Uint16(fmtData[2:4]))
			sampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
			byteRate = int(binary.LittleEndian.Uint32(fmtData[8:12]))
			blockAlign = int(binary.LittleEndian.Uint16(fmtData[12:14]))
			bitsPerSample = int(binary.LittleEndian.Uint16(fmtData[14:16]))
		case "data":
			if dataBytes < 0 {
				dataBytes = chunkSize
			}
		}
		next := chunkEnd
		if next%2 != 0 {
			next++
		}
		if next > int64(len(data)) {
			return WAVMetadata{}, errors.New("WAV chunk padding is invalid")
		}
		offset = int(next)
	}
	if sampleRate <= 0 || sampleRate > 384000 || channels <= 0 || channels > 32 || bitsPerSample != 16 || dataBytes < 0 {
		return WAVMetadata{}, errors.New("WAV must contain 16-bit PCM format and data chunks")
	}
	expectedBlockAlign := channels * bitsPerSample / 8
	if expectedBlockAlign <= 0 || blockAlign != expectedBlockAlign || byteRate != sampleRate*blockAlign || dataBytes%int64(blockAlign) != 0 {
		return WAVMetadata{}, errors.New("WAV PCM format fields are inconsistent")
	}
	if dataBytes == 0 {
		return WAVMetadata{}, errors.New("WAV audio data is empty")
	}
	durationSeconds := float64(dataBytes) / float64(byteRate)
	if math.IsNaN(durationSeconds) || math.IsInf(durationSeconds, 0) || durationSeconds <= 0 {
		return WAVMetadata{}, errors.New("WAV duration is invalid")
	}
	duration := time.Duration(durationSeconds * float64(time.Second))
	if duration > QwenAudioMaxDuration {
		return WAVMetadata{}, fmt.Errorf("WAV duration exceeds %d seconds", int(QwenAudioMaxDuration.Seconds()))
	}
	return WAVMetadata{
		SampleRate: sampleRate, Channels: channels, BitsPerSample: bitsPerSample,
		DataBytes: dataBytes, Duration: duration,
	}, nil
}

type QwenASRResult struct {
	Text          string
	RequestID     string
	UpstreamModel string
	Duration      time.Duration
}

type QwenTTSRequest struct {
	Model        string
	Input        string
	Voice        string
	Speed        float64
	Instructions string
}

type QwenTTSResult struct {
	Audio            []byte
	TaskID           string
	UpstreamModel    string
	BilledCharacters int
	Duration         time.Duration
}

// ResolveQwenAudioModel requires a real account-level mapping match.  This
// prevents an unconfigured generic OpenAI account from receiving a Qwen native
// request merely because an empty mapping normally means "allow all".
func ResolveQwenAudioModel(account *Account, requestedModel string) (string, error) {
	if account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeAPIKey {
		return "", errors.New("qwen audio requires an OpenAI API-key account")
	}
	mapped, matched := account.ResolveMappedModel(strings.TrimSpace(requestedModel))
	if !matched || strings.TrimSpace(mapped) == "" {
		return "", errors.New("model is not explicitly mapped for Qwen audio")
	}
	return strings.TrimSpace(mapped), nil
}

func (s *OpenAIGatewayService) ForwardQwenASR(ctx context.Context, account *Account, requestedModel string, wav []byte, meta WAVMetadata) (*QwenASRResult, error) {
	if s == nil || s.httpUpstream == nil || account == nil {
		return nil, errors.New("qwen ASR service is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	upstreamModel, err := ResolveQwenAudioModel(account, requestedModel)
	if err != nil {
		return nil, err
	}
	if upstreamModel != "qwen-audio-3.0-asr-flash" {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	if err := validateQwenAudioEndpointPair(account); err != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	validatedMeta, err := ParsePCM16WAV(wav)
	if err != nil {
		return nil, err
	}
	// Do not trust caller-supplied metadata when constructing the upstream
	// request. The handler passes its preflight result, but the service remains
	// safe for every other caller as well.
	meta = validatedMeta
	target, err := qwenAudioEndpoint(account, QwenAudioHTTPBaseURLExtraKey, "https", qwenASRPath)
	if err != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	token, _, err := s.getRequestCredential(ctx, nil, account)
	if err != nil || strings.TrimSpace(token) == "" {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	payload := map[string]any{
		"model": upstreamModel,
		"input": map[string]any{"messages": []any{map[string]any{
			"role": "user",
			"content": []any{map[string]any{
				"type": "input_audio",
				"input_audio": map[string]any{
					"data": "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(wav),
				},
			}},
		}}},
		"parameters": map[string]any{
			"format":      "wav",
			"sample_rate": strconv.Itoa(meta.SampleRate),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, qwenASRTotalTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-DashScope-SSE", "disable")
	account.ApplyHeaderOverrides(req.Header)

	started := time.Now()
	resp, err := s.httpUpstream.Do(req, resolveAccountProxyURL(account), account.ID, account.Concurrency)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, qwenASRResponseMaxBytes+1))
	if readErr != nil || len(responseBody) > qwenASRResponseMaxBytes {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, resp.Header)
	}
	if resp.StatusCode >= 400 {
		return nil, qwenAudioStatusError(resp.StatusCode, resp.Header)
	}
	var decoded struct {
		RequestID string `json:"request_id"`
		Output    struct {
			Text     string `json:"text"`
			Sentence struct {
				Text string `json:"text"`
			} `json:"sentence"`
			Output struct {
				Text     string `json:"text"`
				Sentence struct {
					Text string `json:"text"`
				} `json:"sentence"`
			} `json:"output"`
		} `json:"output"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, resp.Header)
	}
	text := firstNonEmpty(decoded.Output.Text, decoded.Output.Sentence.Text, decoded.Output.Output.Text, decoded.Output.Output.Sentence.Text)
	if strings.TrimSpace(text) == "" {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, resp.Header)
	}
	return &QwenASRResult{
		Text: strings.TrimSpace(text), RequestID: strings.TrimSpace(decoded.RequestID),
		UpstreamModel: upstreamModel, Duration: time.Since(started),
	}, nil
}

type qwenAudioFrameReader interface {
	ReadFrame(context.Context) (coderws.MessageType, []byte, error)
}

type qwenTTSServerEvent struct {
	Header struct {
		TaskID       string `json:"task_id"`
		Event        string `json:"event"`
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	} `json:"header"`
	Payload struct {
		Usage struct {
			Characters *int `json:"characters"`
		} `json:"usage"`
	} `json:"payload"`
}

func (s *OpenAIGatewayService) ForwardQwenTTS(ctx context.Context, account *Account, in QwenTTSRequest) (*QwenTTSResult, error) {
	if s == nil || account == nil {
		return nil, errors.New("qwen TTS service is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !utf8.ValidString(in.Input) || strings.TrimSpace(in.Input) == "" || utf8.RuneCountInString(in.Input) > QwenAudioMaxTTSCharacters {
		return nil, errors.New("qwen TTS input is invalid")
	}
	if !utf8.ValidString(in.Instructions) || utf8.RuneCountInString(in.Instructions) > QwenAudioMaxInstructionCharacters || strings.TrimSpace(in.Voice) == "" || math.IsNaN(in.Speed) || math.IsInf(in.Speed, 0) || in.Speed < 0.5 || in.Speed > 2.0 {
		return nil, errors.New("qwen TTS parameters are invalid")
	}
	upstreamModel, err := ResolveQwenAudioModel(account, in.Model)
	if err != nil {
		return nil, err
	}
	if upstreamModel != "qwen-audio-3.0-tts-plus" {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	if err := validateQwenAudioEndpointPair(account); err != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	target, err := qwenAudioEndpoint(account, QwenAudioWSBaseURLExtraKey, "wss", qwenTTSPath)
	if err != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	token, _, err := s.getRequestCredential(ctx, nil, account)
	if err != nil || strings.TrimSpace(token) == "" {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	dialer := s.getOpenAIWSPassthroughDialer()
	if dialer == nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	dialCtx, dialCancel := context.WithTimeout(ctx, qwenAudioDialTimeout)
	headers := http.Header{"Authorization": []string{"Bearer " + token}}
	account.ApplyHeaderOverrides(headers)
	conn, status, responseHeaders, err := dialer.Dial(dialCtx, target, headers, resolveAccountProxyURL(account))
	dialCancel()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if status <= 0 {
			status = http.StatusBadGateway
		}
		return nil, qwenAudioStatusError(status, responseHeaders)
	}
	defer func() { _ = conn.Close() }()
	frameConn, ok := conn.(qwenAudioFrameReader)
	if !ok {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}

	taskID := uuid.NewString()
	parameters := map[string]any{
		"text_type": "PlainText", "voice": in.Voice, "format": "mp3", "rate": in.Speed,
	}
	if strings.TrimSpace(in.Instructions) != "" {
		parameters["instruction"] = in.Instructions
	}
	runTask := map[string]any{
		"header": map[string]any{"action": "run-task", "task_id": taskID, "streaming": "duplex"},
		"payload": map[string]any{
			"task_group": "audio", "task": "tts", "function": "SpeechSynthesizer", "model": upstreamModel,
			"parameters": parameters,
			"input":      map[string]any{},
		},
	}
	started := time.Now()
	totalCtx, totalCancel := context.WithTimeout(ctx, qwenTTSTotalTimeout)
	defer totalCancel()
	if err := conn.WriteJSON(totalCtx, runTask); err != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}

	firstCtx, firstCancel := context.WithTimeout(totalCtx, qwenAudioFirstEventWait)
	msgType, frame, err := frameConn.ReadFrame(firstCtx)
	firstCancel()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	event, eventErr := parseQwenTTSEvent(msgType, frame, taskID)
	if eventErr != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	if event.Header.Event == "task-failed" {
		return nil, qwenTTSTaskFailure(event.Header.ErrorCode)
	}
	if event.Header.Event != "task-started" {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	continueTask := map[string]any{
		"header":  map[string]any{"action": "continue-task", "task_id": taskID, "streaming": "duplex"},
		"payload": map[string]any{"input": map[string]any{"text": in.Input}},
	}
	finishTask := map[string]any{
		"header":  map[string]any{"action": "finish-task", "task_id": taskID, "streaming": "duplex"},
		"payload": map[string]any{"input": map[string]any{}},
	}
	if err := conn.WriteJSON(totalCtx, continueTask); err != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
	if err := conn.WriteJSON(totalCtx, finishTask); err != nil {
		return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
	}

	var audio bytes.Buffer
	for {
		msgType, frame, err = frameConn.ReadFrame(totalCtx)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
		}
		if msgType == coderws.MessageBinary {
			if audio.Len()+len(frame) > qwenTTSOutputMaxBytes {
				return nil, errQwenAudioOutputTooLarge
			}
			_, _ = audio.Write(frame)
			continue
		}
		event, eventErr = parseQwenTTSEvent(msgType, frame, taskID)
		if eventErr != nil {
			return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
		}
		switch event.Header.Event {
		case "result-generated":
			continue
		case "task-failed":
			return nil, qwenTTSTaskFailure(event.Header.ErrorCode)
		case "task-finished":
			if event.Payload.Usage.Characters == nil || *event.Payload.Usage.Characters <= 0 || *event.Payload.Usage.Characters > qwenAudioProviderMaxTaskCharacters || !looksLikeMP3(audio.Bytes()) {
				return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
			}
			return &QwenTTSResult{
				Audio: audio.Bytes(), TaskID: taskID, UpstreamModel: upstreamModel,
				BilledCharacters: *event.Payload.Usage.Characters, Duration: time.Since(started),
			}, nil
		default:
			return nil, qwenAudioRetryableError(http.StatusBadGateway, nil)
		}
	}
}

func parseQwenTTSEvent(messageType coderws.MessageType, payload []byte, taskID string) (qwenTTSServerEvent, error) {
	if messageType != coderws.MessageText {
		return qwenTTSServerEvent{}, errQwenAudioProtocol
	}
	var event qwenTTSServerEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return qwenTTSServerEvent{}, errQwenAudioProtocol
	}
	if strings.TrimSpace(event.Header.Event) == "" || event.Header.TaskID != taskID {
		return qwenTTSServerEvent{}, errQwenAudioProtocol
	}
	return event, nil
}

func qwenTTSTaskFailure(code string) error {
	normalized := strings.ToLower(strings.TrimSpace(code))
	switch {
	case strings.Contains(normalized, "thrott"), strings.Contains(normalized, "rate"), strings.Contains(normalized, "quota"):
		return qwenAudioStatusError(http.StatusTooManyRequests, nil)
	case strings.Contains(normalized, "invalid"), strings.Contains(normalized, "badrequest"):
		return qwenAudioStatusError(http.StatusBadRequest, nil)
	case strings.Contains(normalized, "auth"), strings.Contains(normalized, "unauthorized"):
		return qwenAudioStatusError(http.StatusUnauthorized, nil)
	default:
		return qwenAudioRetryableError(http.StatusBadGateway, nil)
	}
}

func qwenAudioStatusError(status int, headers http.Header) error {
	if status <= 0 {
		status = http.StatusBadGateway
	}
	action := NextAccountStop
	if status == http.StatusTooManyRequests || status >= 500 {
		action = NextAccountRetry
	}
	return &UpstreamFailoverError{
		StatusCode: status, ResponseHeaders: cloneHeader(headers),
		Scope: GatewayFailureScopeProvider, NextAccountAction: action,
	}
}

func qwenAudioRetryableError(status int, headers http.Header) error {
	if status <= 0 {
		status = http.StatusBadGateway
	}
	return &UpstreamFailoverError{
		StatusCode: status, ResponseHeaders: cloneHeader(headers),
		Scope: GatewayFailureScopeProvider, NextAccountAction: NextAccountRetry,
	}
}

func qwenAudioEndpoint(account *Account, extraKey, scheme, requiredPath string) (string, error) {
	if account == nil {
		return "", errors.New("qwen audio account is required")
	}
	raw := strings.TrimSpace(account.GetExtraString(extraKey))
	if raw == "" {
		return "", fmt.Errorf("%s is required", extraKey)
	}
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, scheme) || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("%s must be a secure absolute URL", extraKey)
	}
	hostname := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if !isQwenAudioWorkspaceHost(hostname) {
		return "", fmt.Errorf("%s host must be a supported workspace Maas endpoint", extraKey)
	}
	if port := u.Port(); port != "" && port != "443" {
		return "", fmt.Errorf("%s must use the default TLS port", extraKey)
	}
	cleanPath := path.Clean("/" + strings.TrimSpace(u.Path))
	requiredPath = path.Clean("/" + strings.TrimSpace(requiredPath))
	if cleanPath == "." || cleanPath == "/" {
		cleanPath = requiredPath
	} else if !strings.HasSuffix(cleanPath, requiredPath) {
		switch {
		case cleanPath == "/api/v1" && strings.HasPrefix(requiredPath, "/api/v1/"):
			cleanPath = requiredPath
		case cleanPath == "/api-ws/v1" && strings.HasPrefix(requiredPath, "/api-ws/v1/"):
			cleanPath = requiredPath
		default:
			cleanPath = strings.TrimSuffix(cleanPath, "/") + requiredPath
		}
	}
	u.Path = cleanPath
	u.RawPath = ""
	return u.String(), nil
}

func isQwenAudioWorkspaceHost(hostname string) bool {
	parts := strings.Split(strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), ".")), ".")
	if len(parts) != 5 || parts[2] != "maas" || parts[3] != "aliyuncs" || parts[4] != "com" {
		return false
	}
	if parts[1] != "cn-beijing" && parts[1] != "ap-southeast-1" {
		return false
	}
	workspace := parts[0]
	if workspace == "" || workspace[0] == '-' || workspace[len(workspace)-1] == '-' {
		return false
	}
	for _, char := range workspace {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func validateQwenAudioEndpointPair(account *Account) error {
	httpEndpoint, err := qwenAudioEndpoint(account, QwenAudioHTTPBaseURLExtraKey, "https", qwenASRPath)
	if err != nil {
		return err
	}
	wsEndpoint, err := qwenAudioEndpoint(account, QwenAudioWSBaseURLExtraKey, "wss", qwenTTSPath)
	if err != nil {
		return err
	}
	httpURL, _ := url.Parse(httpEndpoint)
	wsURL, _ := url.Parse(wsEndpoint)
	if !strings.EqualFold(httpURL.Hostname(), wsURL.Hostname()) {
		return errors.New("qwen audio HTTP and WebSocket endpoints must use the same workspace host")
	}
	return nil
}

func accountHasSupportedQwenAudioMapping(account *Account) bool {
	if account == nil {
		return false
	}
	for _, upstreamModel := range account.GetModelMapping() {
		switch strings.TrimSpace(upstreamModel) {
		case "qwen-audio-3.0-asr-flash", "qwen-audio-3.0-tts-plus":
			return true
		}
	}
	return false
}

func looksLikeMP3(data []byte) bool {
	if len(data) >= 3 && string(data[:3]) == "ID3" {
		return true
	}
	return len(data) >= 2 && data[0] == 0xff && data[1]&0xe0 == 0xe0
}
