package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/google/uuid"
)

func chatDocumentExtensions(native, ocr bool) []string {
	result := []string{}
	for _, ext := range strings.Fields(chatUploadExtensions) {
		if native {
			result = append(result, ext)
			continue
		}
		_, purpose, _ := chatFileKind("file." + ext)
		if purpose == "image" || ext == "xls" || ext == "xlsx" || ext == "epub" || ext == "svgz" {
			continue
		}
		if purpose == "rich_layout_recognition" && !ocr {
			continue
		}
		result = append(result, ext)
	}
	return result
}

func (s *ChatService) localDocumentText(ctx context.Context, user int64, id uuid.UUID) (string, error) {
	record, file, err := s.OpenAttachment(ctx, user, id)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if record.Purpose != "user_data" {
		return "", ErrChatCapability
	}
	raw, err := readChatBounded(file, ChatMaxParseBytes)
	if err != nil {
		return "", err
	}
	text, err := decodeChatDocumentText(raw)
	if err != nil {
		return "", err
	}
	return "[附件文本：" + record.Name + "]\n" + text, nil
}

func decodeChatDocumentText(raw []byte) (string, error) {
	if bytes.HasPrefix(raw, []byte{0xff, 0xfe}) || bytes.HasPrefix(raw, []byte{0xfe, 0xff}) {
		if len(raw)%2 != 0 {
			return "", ErrChatInvalid
		}
		var order binary.ByteOrder = binary.LittleEndian
		if raw[0] == 0xfe {
			order = binary.BigEndian
		}
		units := make([]uint16, (len(raw)-2)/2)
		for i := range units {
			units[i] = order.Uint16(raw[2+i*2 : 4+i*2])
		}
		for i := 0; i < len(units); i++ {
			unit := units[i]
			if unit >= 0xd800 && unit <= 0xdbff {
				if i+1 >= len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
					return "", ErrChatInvalid
				}
				i++
			} else if unit >= 0xdc00 && unit <= 0xdfff {
				return "", ErrChatInvalid
			}
			if unit == 0 {
				return "", ErrChatInvalid
			}
		}
		return string(utf16.Decode(units)), nil
	}
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		return "", ErrChatCapability
	}
	return string(raw), nil
}

// Called only after the generation lease is held. Uploading or previewing a
// document never starts OCR; durable claims make repeated sends reuse its result.
func (s *ChatService) prepareChatDocuments(ctx context.Context, run *ChatRunRecord, clientIP string, documents map[uuid.UUID]bool) error {
	for id := range documents {
		if err := ctx.Err(); err != nil {
			return err
		}
		record, err := s.ParseAttachment(ctx, run.UserID, id, run.PresetID, run.SessionID, clientIP, run.Operation == "regenerate")
		if err != nil {
			return err
		}
		wait, cancel := context.WithTimeout(ctx, 4*time.Minute)
		for record.Status == "processing" || record.Status == "settling" {
			timer := time.NewTimer(250 * time.Millisecond)
			select {
			case <-wait.Done():
				timer.Stop()
				cancel()
				return ErrChatParseBusy
			case <-timer.C:
			}
			record, err = s.GetParsedAttachment(wait, run.UserID, id, run.PresetID)
			if err != nil {
				cancel()
				return err
			}
		}
		cancel()
		if record.Status == "unknown" {
			return ErrChatParseUnknown
		}
		if record.Status != "ready" || !record.BillingApplied {
			return ErrChatParseRequired
		}
	}
	return nil
}
