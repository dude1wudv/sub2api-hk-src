package service

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type ChatAttachmentRecord struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Size       int64      `json:"size"`
	MIME       string     `json:"mimeType"`
	IsImage    bool       `json:"isImage"`
	UserID     int64      `json:"-"`
	StorageKey string     `json:"-"`
	SHA256     string     `json:"-"`
	Purpose    string     `json:"-"`
	Status     string     `json:"-"`
	ImportID   *uuid.UUID `json:"-"`
}

type ChatAttachmentRepository interface {
	ReserveAttachment(context.Context, *ChatAttachmentRecord, int64) error
	FinishAttachment(context.Context, *ChatAttachmentRecord) error
	AbandonAttachment(context.Context, int64, uuid.UUID) error
	GetAttachment(context.Context, int64, uuid.UUID) (*ChatAttachmentRecord, error)
}

var chatExtensionSet = func() map[string]bool {
	out := map[string]bool{}
	for _, ext := range strings.Fields(chatUploadExtensions) {
		out[ext] = true
	}
	return out
}()

func IsChatPreviewMIME(mime string) bool {
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp", "image/tiff", "image/avif", "image/x-icon":
		return true
	}
	return false
}

func chatFileKind(name string) (string, string, error) {
	if name == "" || len(name) > 255 || !utf8.ValidString(name) || strings.ContainsAny(name, "/\\:\x00\r\n") || strings.TrimSpace(name) != name {
		return "", "", ErrChatInvalid
	}
	for _, r := range name {
		if r < 32 || r == 127 {
			return "", "", ErrChatInvalid
		}
	}
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(name)), ".")
	if !chatExtensionSet[ext] {
		return "", "", ErrChatCapability
	}
	purpose := "user_data"
	switch ext {
	case "png", "apng", "jpg", "jpeg", "jfif", "pjp", "pjpeg", "gif", "webp", "bmp", "dib", "tif", "tiff", "avif", "ico":
		purpose = "image"
	case "pdf", "doc", "docx", "ppt", "pptx":
		purpose = "rich_layout_recognition"
	}
	return ext, purpose, nil
}

func chatValidateZipFile(source io.ReadSeeker, size int64, ext string) error {
	reader, ok := source.(io.ReaderAt)
	if !ok {
		return ErrChatInvalid
	}
	archive, err := zip.NewReader(reader, size)
	if err != nil || len(archive.File) > 10000 {
		return ErrChatInvalid
	}
	required := map[string]string{"docx": "word/document.xml", "pptx": "ppt/presentation.xml", "xlsx": "xl/workbook.xml", "epub": "META-INF/container.xml"}[ext]
	found, types := false, false
	seen := map[string]bool{}
	var total uint64
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.HasPrefix(name, "../") || seen[name] || file.Mode()&os.ModeSymlink != 0 {
			return ErrChatInvalid
		}
		seen[name] = true
		if file.UncompressedSize64 > 256<<20 || total > 256<<20-file.UncompressedSize64 {
			return ErrChatTooLarge
		}
		total += file.UncompressedSize64
		if file.UncompressedSize64 > 1<<20 && (file.CompressedSize64 == 0 || file.UncompressedSize64/file.CompressedSize64 > 1000) {
			return ErrChatTooLarge
		}
		content, err := file.Open()
		if err != nil {
			return ErrChatInvalid
		}
		actual, readErr := io.Copy(io.Discard, io.LimitReader(content, int64(file.UncompressedSize64)+1))
		closeErr := content.Close()
		if readErr != nil || closeErr != nil || uint64(actual) != file.UncompressedSize64 {
			return ErrChatInvalid
		}
		found = found || name == required
		types = types || name == "[Content_Types].xml"
	}
	if !found || (ext != "epub" && !types) {
		return ErrChatInvalid
	}
	return nil
}

func chatValidateSVG(raw []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	depth := 0
	root := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ErrChatInvalid
		}
		switch item := token.(type) {
		case xml.StartElement:
			if !root {
				if item.Name.Local != "svg" {
					return ErrChatInvalid
				}
				root = true
			}
			depth++
			if depth > 256 {
				return ErrChatTooLarge
			}
		case xml.EndElement:
			depth--
		}
	}
	if !root {
		return ErrChatInvalid
	}
	return nil
}

func chatValidateFile(source io.ReadSeeker, size int64, ext string) (string, error) {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	head := make([]byte, min(size, int64(512)))
	if _, err := io.ReadFull(source, head); err != nil {
		return "", err
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	sniff := http.DetectContentType(head)
	switch ext {
	case "png", "apng", "jpg", "jpeg", "jfif", "pjp", "pjpeg", "gif":
		expected := "image/jpeg"
		if ext == "png" || ext == "apng" {
			expected = "image/png"
		}
		if ext == "gif" {
			expected = "image/gif"
		}
		if sniff != expected {
			return "", ErrChatInvalid
		}
		config, _, err := image.DecodeConfig(source)
		if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 100000000 {
			return "", ErrChatInvalid
		}
		return expected, nil
	case "webp":
		if sniff != "image/webp" {
			return "", ErrChatInvalid
		}
		return sniff, nil
	case "bmp", "dib":
		if !bytes.HasPrefix(head, []byte("BM")) {
			return "", ErrChatInvalid
		}
		return "image/bmp", nil
	case "tif", "tiff":
		if !bytes.HasPrefix(head, []byte{'I', 'I', 42, 0}) && !bytes.HasPrefix(head, []byte{'M', 'M', 0, 42}) {
			return "", ErrChatInvalid
		}
		return "image/tiff", nil
	case "avif":
		if len(head) < 16 || string(head[4:8]) != "ftyp" || (!bytes.Contains(head[8:], []byte("avif")) && !bytes.Contains(head[8:], []byte("avis"))) {
			return "", ErrChatInvalid
		}
		return "image/avif", nil
	case "ico":
		if !bytes.HasPrefix(head, []byte{0, 0, 1, 0}) || len(head) < 6 {
			return "", ErrChatInvalid
		}
		return "image/x-icon", nil
	case "pdf":
		if !bytes.HasPrefix(head, []byte("%PDF-")) {
			return "", ErrChatInvalid
		}
		return "application/pdf", nil
	case "doc", "ppt", "xls":
		if !bytes.HasPrefix(head, []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}) {
			return "", ErrChatInvalid
		}
		return map[string]string{"doc": "application/msword", "ppt": "application/vnd.ms-powerpoint", "xls": "application/vnd.ms-excel"}[ext], nil
	case "docx", "pptx", "xlsx", "epub":
		if err := chatValidateZipFile(source, size, ext); err != nil {
			return "", err
		}
		return map[string]string{"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "epub": "application/epub+zip"}[ext], nil
	case "mobi":
		if len(head) < 68 || string(head[60:68]) != "BOOKMOBI" {
			return "", ErrChatInvalid
		}
		return "application/x-mobipocket-ebook", nil
	case "psd":
		if !bytes.HasPrefix(head, []byte("8BPS")) {
			return "", ErrChatInvalid
		}
		return "image/vnd.adobe.photoshop", nil
	}
	var reader io.Reader = source
	if ext == "svgz" {
		gzipReader, err := gzip.NewReader(source)
		if err != nil {
			return "", ErrChatInvalid
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	raw, err := io.ReadAll(io.LimitReader(reader, ChatMaxUserDataBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(raw)) > ChatMaxUserDataBytes {
		return "", ErrChatTooLarge
	}
	if ext == "svg" || ext == "svgz" {
		if err := chatValidateSVG(raw); err != nil {
			return "", err
		}
		if ext == "svgz" {
			return "application/gzip", nil
		}
		return "image/svg+xml", nil
	}
	if (bytes.HasPrefix(raw, []byte{0xff, 0xfe}) || bytes.HasPrefix(raw, []byte{0xfe, 0xff})) && len(raw)%2 == 0 {
		return "text/plain; charset=utf-16", nil
	}
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		return "", ErrChatInvalid
	}
	if !strings.HasPrefix(sniff, "text/") && sniff != "application/xml" && len(raw) > 0 {
		return "", ErrChatInvalid
	}
	return sniff, nil
}

func (s *ChatService) UploadAttachment(ctx context.Context, userID int64, name, purpose string, source io.ReadSeeker, size int64, importID *uuid.UUID) (*ChatAttachmentRecord, error) {
	ext, expected, err := chatFileKind(name)
	if err != nil {
		return nil, err
	}
	if purpose != "" && purpose != expected {
		return nil, ErrChatInvalid
	}
	limit := ChatMaxFileBytes
	if expected == "user_data" {
		limit = ChatMaxUserDataBytes
	}
	if size < 0 || size > limit {
		return nil, ErrChatTooLarge
	}
	mime, err := chatValidateFile(source, size, ext)
	if err != nil {
		return nil, err
	}
	if importID != nil && !IsChatPreviewMIME(mime) {
		return nil, ErrChatInvalid
	}
	if _, err = source.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	id := uuid.New()
	key, err := chatStorageKey(userID, id)
	if err != nil {
		return nil, err
	}
	record := &ChatAttachmentRecord{ID: id, UserID: userID, Name: name, Size: size, MIME: mime, IsImage: IsChatPreviewMIME(mime), StorageKey: key, Purpose: expected, ImportID: importID, Status: "uploading"}
	if err = s.repo.ReserveAttachment(ctx, record, s.storageLimit); err != nil {
		return nil, err
	}
	stored, err := s.storage.WriteFile(ctx, userID, id, source, limit)
	if err == nil && stored.Size != size {
		err = ErrChatInvalid
	}
	if err != nil {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = s.repo.AbandonAttachment(cleanup, userID, id)
		_ = s.storage.RemoveFile(userID, id)
		return nil, err
	}
	record.SHA256 = stored.SHA256
	record.Status = "ready"
	if importID != nil {
		record.Status = "staged"
	}
	if err = s.repo.FinishAttachment(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *ChatService) OpenAttachment(ctx context.Context, userID int64, id uuid.UUID) (*ChatAttachmentRecord, *os.File, error) {
	record, err := s.repo.GetAttachment(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	file, err := s.storage.OpenFile(userID, id)
	if err != nil {
		return nil, nil, ErrChatNotFound
	}
	return record, file, nil
}
