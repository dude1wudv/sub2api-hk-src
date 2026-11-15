package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func validDeepSeekSuiteJSON(raw []byte) bool {
	if !utf8.Valid(raw) || !gjson.ValidBytes(raw) || !gjson.ParseBytes(raw).IsObject() {
		return false
	}
	var validate func(gjson.Result, int) bool
	validate = func(value gjson.Result, depth int) bool {
		if depth > 128 {
			return false
		}
		if !value.IsArray() && !value.IsObject() {
			return true
		}
		seen, valid := map[string]bool{}, true
		value.ForEach(func(key, child gjson.Result) bool {
			if value.IsObject() {
				if seen[key.String()] {
					valid = false
					return false
				}
				seen[key.String()] = true
			}
			valid = validate(child, depth+1)
			return valid
		})
		return valid
	}
	return validate(gjson.ParseBytes(raw), 0)
}

const DeepSeekSuiteMaxUploadBytes int64 = 100<<20 + 64<<10

type DeepSeekSuiteInput struct {
	Kind           string
	Body           []byte
	ContentType    string
	File           *multipart.FileHeader
	Fields         map[string]string
	ReserveCost    float64
	AuditText      string
	ResponseFormat string
	AnthropicFiles bool
	Conversation   bool
	Stream         bool
	Model          string
	TTSCharacters  int64
}

func ParseDeepSeekSuiteInput(request *http.Request, kind string, audioCompat bool) (*DeepSeekSuiteInput, error) {
	input := &DeepSeekSuiteInput{Kind: kind, ContentType: request.Header.Get("Content-Type"),
		Fields: map[string]string{}, AnthropicFiles: strings.HasPrefix(request.URL.Path, "/anthropic/")}
	mediaType, _, err := mime.ParseMediaType(input.ContentType)
	if err != nil {
		return nil, ErrChatInvalid
	}
	if kind == "ocr" || kind == "asr" || kind == "files" {
		if mediaType != "multipart/form-data" {
			return nil, errors.New("multipart/form-data with one file is required")
		}
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			return nil, err
		}
		files := request.MultipartForm.File["file"]
		if len(files) != 1 || len(request.MultipartForm.File) != 1 {
			return nil, errors.New("exactly one file is required")
		}
		input.File = files[0]
		input.File.Filename = path.Base(strings.ReplaceAll(input.File.Filename, `\`, "/"))
		if len(input.File.Filename) > 255 || strings.ContainsAny(input.File.Filename, "\r\n\x00") {
			return nil, ErrChatInvalid
		}
		limit := int64(100 << 20)
		if kind == "asr" {
			limit = 8 << 20
		}
		if input.File.Size <= 0 || input.File.Size > limit {
			return nil, ErrChatTooLarge
		}
		for k, values := range request.MultipartForm.Value {
			if len(values) != 1 || len(values[0]) > 4096 {
				return nil, ErrChatInvalid
			}
			input.Fields[k] = values[0]
		}
		if audioCompat {
			if input.Fields["model"] != "deepseek-asr" {
				return nil, ErrChatInvalid
			}
			input.ResponseFormat = input.Fields["response_format"]
			if input.ResponseFormat != "" && input.ResponseFormat != "json" && input.ResponseFormat != "text" {
				return nil, errors.New("DeepSeek ASR supports json and text response formats")
			}
			delete(input.Fields, "model")
			delete(input.Fields, "response_format")
			if len(input.Fields) != 0 {
				return nil, errors.New("DeepSeek ASR does not support extra transcription options")
			}
		}
		input.ReserveCost = 1.5
		if kind == "asr" {
			input.ReserveCost = .78 / 30 // the maximum 120-second recording
		}
		if kind == "files" {
			purpose := input.Fields["purpose"]
			if purpose == "" && input.AnthropicFiles {
				purpose = "user_data"
				input.Fields["purpose"] = purpose
			}
			if purpose != "user_data" && purpose != "text_recognition" && purpose != "rich_layout_recognition" {
				return nil, errors.New("unsupported file purpose")
			}
			if purpose == "user_data" {
				input.ReserveCost = 0
			}
		}
		for name := range input.Fields {
			if (kind == "asr") || name != "purpose" {
				return nil, errors.New("unsupported multipart field: " + name)
			}
		}
		if kind == "ocr" && input.Fields["purpose"] != "text_recognition" && input.Fields["purpose"] != "rich_layout_recognition" {
			return nil, errors.New("OCR requires text_recognition or rich_layout_recognition")
		}
		if kind != "asr" {
			file, openErr := input.File.Open()
			if openErr != nil {
				return nil, openErr
			}
			var header [512]byte
			n, _ := file.Read(header[:])
			_ = file.Close()
			mimeType := http.DetectContentType(header[:n])
			extension := strings.ToLower(path.Ext(input.File.Filename))
			document := mimeType == "application/pdf" || extension == ".pdf" || extension == ".doc" || extension == ".docx" || extension == ".ppt" || extension == ".pptx"
			if kind == "files" && document && input.Fields["purpose"] == "user_data" {
				return nil, errors.New("PDF, Word and PowerPoint require a document parsing purpose")
			}
			maxSize := int64(32 << 20)
			if document {
				maxSize = 100 << 20
			} else if strings.HasPrefix(mimeType, "image/") {
				maxSize = 64 << 20
			}
			if input.File.Size > maxSize {
				return nil, ErrChatTooLarge
			}
		}
		return input, nil
	}
	if mediaType != "application/json" {
		return nil, errors.New("Content-Type must be application/json")
	}
	limit := int64(64 << 10)
	input.Conversation = kind == "chat" || kind == "responses" || kind == "messages"
	if input.Conversation {
		limit = 32 << 20
	}
	input.Body, err = readChatBounded(request.Body, limit)
	if err != nil || !validDeepSeekSuiteJSON(input.Body) {
		return nil, ErrChatInvalid
	}
	if input.Conversation {
		input.Model = gjson.GetBytes(input.Body, "model").String()
		if strings.TrimSpace(input.Model) == "" {
			return nil, errors.New("model is required")
		}
		if safety := gjson.GetBytes(input.Body, "safety_check"); safety.Exists() {
			if level := safety.Get("level").String(); level != "high" && level != "low" {
				return nil, errors.New("safety_check.level must be high or low")
			}
		}
		if kind == "responses" && bytes.Contains(input.Body, []byte(`"latest_reminder"`)) {
			return nil, errors.New("latest_reminder is not supported by Responses")
		}
		input.Stream = gjson.GetBytes(input.Body, "stream").Bool()
		input.ReserveCost = .10
		input.AuditText = string(input.Body)
		return input, nil
	}
	if kind == "search" || kind == "fetch" {
		input.AuditText, err = ValidateDeepSeekToolRequest(DeepSeekToolKind(kind), input.Body)
		input.ReserveCost = .005
		return input, err
	}
	if kind != "tts" {
		return nil, ErrChatInvalid
	}
	if audioCompat {
		var m map[string]json.RawMessage
		if json.Unmarshal(input.Body, &m) != nil || gjson.GetBytes(input.Body, "model").String() != "deepseek-tts" {
			return nil, ErrChatInvalid
		}
		for k := range m {
			switch k {
			case "model", "input", "voice", "response_format", "speed":
			default:
				return nil, errors.New("unsupported DeepSeek speech option: " + k)
			}
		}
		if speed := gjson.GetBytes(input.Body, "speed"); speed.Exists() && speed.Float() != 1 {
			return nil, errors.New("DeepSeek does not support changing speech speed")
		}
		m["text"], m["voice_id"], m["format"] = m["input"], m["voice"], m["response_format"]
		for _, k := range []string{"model", "input", "voice", "response_format", "speed"} {
			delete(m, k)
		}
		for k, v := range m {
			if len(v) == 0 {
				delete(m, k)
			}
		}
		input.Body, _ = json.Marshal(m)
	}
	if gjson.GetBytes(input.Body, "text").Type != gjson.String {
		return nil, ErrChatInvalid
	}
	if value := gjson.GetBytes(input.Body, "filter_markdown"); value.Exists() && value.Type != gjson.True && value.Type != gjson.False {
		return nil, ErrChatInvalid
	}
	text := gjson.GetBytes(input.Body, "text").String()
	voice := gjson.GetBytes(input.Body, "voice_id").String()
	format := gjson.GetBytes(input.Body, "format").String()
	if format != "" && format != "wav" && format != "pcm" {
		return nil, errors.New("DeepSeek HTTP TTS supports wav or pcm")
	}
	input.TTSCharacters, err = ValidateChatTTS(&ChatTTSInput{Text: text, VoiceID: voice})
	if err != nil {
		return nil, err
	}
	input.AuditText, input.Stream = text, true
	usage, _ := chatAuxiliaryUsage("tts", &input.TTSCharacters, "confirmed")
	input.ReserveCost, _ = usage.cost()
	return input, nil
}

func (in *DeepSeekSuiteInput) upstreamBody(op *DeepSeekSuiteOperation) (io.ReadCloser, string, error) {
	if in.File == nil {
		return io.NopCloser(bytes.NewReader(in.Body)), "application/json", nil
	}
	file, err := in.File.Open()
	if err != nil {
		return nil, "", err
	}
	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	go func() {
		defer func() { _ = file.Close() }()
		var err error
		// Official document parsing requires purpose before the file.
		if purpose := in.Fields["purpose"]; purpose != "" {
			err = form.WriteField("purpose", purpose)
		}
		for key, value := range in.Fields {
			if key != "purpose" && err == nil {
				err = form.WriteField(key, value)
			}
		}
		name := in.File.Filename
		if op != nil && op.Kind == "files" {
			name = op.UpstreamName
		}
		if err == nil {
			var part io.Writer
			part, err = form.CreateFormFile("file", name)
			if err == nil {
				_, err = io.Copy(part, file)
			}
		}
		if err == nil {
			err = form.Close()
		}
		_ = writer.CloseWithError(err)
	}()
	return reader, form.FormDataContentType(), nil
}

func suiteFileMetadata(raw []byte, op *DeepSeekSuiteOperation, anthropic bool) ([]byte, error) {
	var err error
	raw, err = sjson.SetBytes(raw, "id", op.FileID())
	if err == nil {
		raw, err = sjson.SetBytes(raw, "filename", op.Filename)
	}
	if err != nil {
		return nil, err
	}
	if anthropic {
		raw, _ = sjson.SetBytes(raw, "type", "file")
		raw, _ = sjson.SetBytes(raw, "size_bytes", op.Bytes)
		raw, _ = sjson.SetBytes(raw, "created_at", op.CreatedAt.Format("2006-01-02T15:04:05Z"))
		raw, _ = sjson.DeleteBytes(raw, "object")
	} else {
		raw, _ = sjson.SetBytes(raw, "object", "file")
		raw, _ = sjson.SetBytes(raw, "bytes", op.Bytes)
		raw, _ = sjson.SetBytes(raw, "created_at", op.CreatedAt.Unix())
		raw, _ = sjson.DeleteBytes(raw, "type")
	}
	return raw, nil
}
