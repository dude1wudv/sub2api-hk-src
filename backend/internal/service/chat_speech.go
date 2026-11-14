package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const ChatMaxSpeechFileBytes = 8 << 20
const ChatMaxSpeechJSONBytes = 64 << 10

// The browser cannot supply upstream URLs, models, accounts, or billing IDs.
type ChatSpeechRequest struct {
	PresetID       uuid.UUID `json:"preset_id"`
	SessionID      uuid.UUID `json:"session_id"`
	Text           string    `json:"text"`
	VoiceID        string    `json:"voice_id,omitempty"`
	FilterMarkdown *bool     `json:"filter_markdown,omitempty"`
}

type ChatTTSInput struct {
	Text           string `json:"text"`
	VoiceID        string `json:"voice_id,omitempty"`
	FilterMarkdown *bool  `json:"filter_markdown,omitempty"`
}

type ChatAuxiliaryInput struct {
	Body        []byte
	ContentType string
	AuditText   string
	ReserveCost float64
	TTS         *ChatTTSInput
}

func readChatBounded(reader io.Reader, limit int64) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, ErrChatTooLarge
	}
	return raw, nil
}

func decodeChatSpeechJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return ErrChatInvalid
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return ErrChatInvalid
	}
	return nil
}

func ValidateChatTTS(input *ChatTTSInput) (int64, error) {
	if input == nil || strings.TrimSpace(input.Text) == "" || len(input.Text) > ChatMaxSpeechJSONBytes {
		return 0, ErrChatInvalid
	}
	if input.VoiceID != "" && input.VoiceID != "stella" && input.VoiceID != "tide" {
		return 0, ErrChatInvalid
	}
	count, err := ChatAuxiliaryTTSCharacters(input.Text)
	if err != nil || count < 1 || count > 10000 {
		return 0, ErrChatInvalid
	}
	return count, nil
}

func ParseChatAuxiliaryInput(kind string, request *http.Request) (*ChatAuxiliaryInput, error) {
	mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		return nil, ErrChatInvalid
	}
	limit := int64(ChatMaxSpeechJSONBytes)
	if kind == "asr" {
		limit = ChatMaxSpeechFileBytes + (64 << 10)
	}
	raw, err := readChatBounded(request.Body, limit)
	if err != nil {
		return nil, err
	}
	input := &ChatAuxiliaryInput{Body: raw, ContentType: mediaType}
	switch kind {
	case "ocr":
		if mediaType != "application/json" {
			return nil, ErrChatInvalid
		}
		var value struct {
			AttachmentID uuid.UUID `json:"attachment_id"`
		}
		if decodeChatSpeechJSON(raw, &value) != nil || value.AttachmentID == uuid.Nil {
			return nil, ErrChatInvalid
		}
		scope, _ := request.Context().Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
		if scope == nil || scope.parse == nil || scope.chat == nil || scope.parse.AttachmentID != value.AttachmentID || scope.operationID != scope.parse.OperationID {
			return nil, ErrChatInvalid
		}
		input.ReserveCost = 1.5
	case "search", "fetch":
		if mediaType != "application/json" {
			return nil, ErrChatInvalid
		}
		input.AuditText, err = ValidateDeepSeekToolRequest(DeepSeekToolKind(kind), raw)
		input.ReserveCost = .005
	case "tts":
		if mediaType != "application/json" {
			return nil, ErrChatInvalid
		}
		input.TTS = &ChatTTSInput{}
		if err = decodeChatSpeechJSON(raw, input.TTS); err != nil {
			return nil, err
		}
		var quantity int64
		quantity, err = ValidateChatTTS(input.TTS)
		if err == nil {
			input.AuditText = input.TTS.Text
			usage, _ := chatAuxiliaryUsage("tts", &quantity, "confirmed")
			input.ReserveCost, _ = usage.cost()
		}
	case "asr":
		if mediaType != "multipart/form-data" || params["boundary"] == "" {
			return nil, ErrChatInvalid
		}
		reader := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
		part, partErr := reader.NextPart()
		if partErr != nil || part.FormName() != "file" || part.FileName() == "" {
			return nil, ErrChatInvalid
		}
		audio, readErr := readChatBounded(part, ChatMaxSpeechFileBytes)
		if readErr != nil {
			return nil, readErr
		}
		if _, partErr = reader.NextPart(); partErr != io.EOF {
			return nil, ErrChatInvalid
		}
		if _, err = ValidateChatSpeechAudio(audio); err != nil {
			return nil, err
		}
		input.ContentType = request.Header.Get("Content-Type")
		// Admission covers the maximum allowed recording. Final billing uses only
		// the provider's authoritative duration, never client or container metadata.
		quantity := int64(120)
		usage, _ := chatAuxiliaryUsage("asr", &quantity, "confirmed")
		input.ReserveCost, _ = usage.cost()
	default:
		return nil, ErrChatInvalid
	}
	if err != nil {
		return nil, err
	}
	return input, nil
}

// ValidateChatSpeechAudio accepts complete PCM WAV or single-stream Ogg Opus.
func ValidateChatSpeechAudio(audio []byte) (string, error) {
	if len(audio) < 12 || len(audio) > ChatMaxSpeechFileBytes {
		return "", ErrChatInvalid
	}
	if string(audio[:4]) == "RIFF" && string(audio[8:12]) == "WAVE" {
		if uint64(binary.LittleEndian.Uint32(audio[4:8]))+8 != uint64(len(audio)) {
			return "", ErrChatInvalid
		}
		var rate, channels, alignment, byteRate uint32
		var dataBytes uint64
		hasFormat, hasData := false, false
		for offset := 12; offset < len(audio); {
			if offset+8 > len(audio) {
				return "", ErrChatInvalid
			}
			kind := string(audio[offset : offset+4])
			size := uint64(binary.LittleEndian.Uint32(audio[offset+4 : offset+8]))
			offset += 8
			if size > uint64(len(audio)-offset) {
				return "", ErrChatInvalid
			}
			chunk := audio[offset : offset+int(size)]
			switch kind {
			case "fmt ":
				if hasFormat || len(chunk) < 16 || binary.LittleEndian.Uint16(chunk[:2]) != 1 || binary.LittleEndian.Uint16(chunk[14:16]) != 16 {
					return "", ErrChatInvalid
				}
				channels = uint32(binary.LittleEndian.Uint16(chunk[2:4]))
				rate = binary.LittleEndian.Uint32(chunk[4:8])
				byteRate = binary.LittleEndian.Uint32(chunk[8:12])
				alignment = uint32(binary.LittleEndian.Uint16(chunk[12:14]))
				hasFormat = true
			case "data":
				if hasData {
					return "", ErrChatInvalid
				}
				hasData = true
				dataBytes = size
			}
			offset += int(size) + (int(size) & 1)
			if offset > len(audio) {
				return "", ErrChatInvalid
			}
		}
		allowedRate := false
		for _, allowed := range []uint32{8000, 16000, 22050, 24000, 32000, 44100, 48000} {
			allowedRate = allowedRate || rate == allowed
		}
		if !hasFormat || !hasData || !allowedRate || channels < 1 || channels > 2 || alignment != channels*2 || byteRate != rate*alignment || dataBytes == 0 || dataBytes%uint64(alignment) != 0 || dataBytes > uint64(byteRate)*120 {
			return "", ErrChatInvalid
		}
		return "wav", nil
	}
	if string(audio[:4]) != "OggS" {
		return "", ErrChatInvalid
	}
	var serial, sequence uint32
	var preSkip uint16
	var granule uint64
	first, ended := true, false
	for offset := 0; offset < len(audio); {
		if ended || offset+27 > len(audio) || string(audio[offset:offset+4]) != "OggS" || audio[offset+4] != 0 {
			return "", ErrChatInvalid
		}
		flags := audio[offset+5]
		segments := int(audio[offset+26])
		header := offset + 27 + segments
		if header > len(audio) {
			return "", ErrChatInvalid
		}
		size := 0
		for _, length := range audio[offset+27 : header] {
			size += int(length)
		}
		if size > len(audio)-header {
			return "", ErrChatInvalid
		}
		currentSerial := binary.LittleEndian.Uint32(audio[offset+14 : offset+18])
		currentSequence := binary.LittleEndian.Uint32(audio[offset+18 : offset+22])
		if first {
			packet := audio[header : header+size]
			if flags&2 == 0 || flags&1 != 0 || len(packet) < 19 || string(packet[:8]) != "OpusHead" || packet[8] != 1 || packet[9] < 1 || packet[9] > 2 {
				return "", ErrChatInvalid
			}
			serial = currentSerial
			sequence = currentSequence
			preSkip = binary.LittleEndian.Uint16(packet[10:12])
			first = false
		} else {
			sequence++
			if currentSerial != serial || currentSequence != sequence || flags&2 != 0 {
				return "", ErrChatInvalid
			}
		}
		granule = binary.LittleEndian.Uint64(audio[offset+6 : offset+14])
		ended = flags&4 != 0
		offset = header + size
	}
	if !ended || granule <= uint64(preSkip) || granule-uint64(preSkip) > 48000*120 {
		return "", ErrChatInvalid
	}
	return "ogg", nil
}

func (s *ChatService) Speech(ctx context.Context, userID int64, presetID, sessionID uuid.UUID, kind, clientIP string, body io.Reader, contentType string, w http.ResponseWriter) error {
	if kind != "asr" && kind != "tts" {
		return ErrChatInvalid
	}
	timeout := 31 * time.Minute
	if kind == "asr" {
		timeout = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	key, scope, err := s.resolveChatAuxiliary(ctx, userID, presetID, sessionID, kind)
	if err != nil {
		return err
	}
	_, err = s.dispatchChatAuxiliary(ctx, w, key, scope, sessionID, clientIP, kind, body, contentType, nil)
	return err
}
