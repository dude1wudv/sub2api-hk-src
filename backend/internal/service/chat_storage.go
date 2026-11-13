package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// ChatStorage never accepts an original filename as a path. os.Root rejects
// traversal and symlink escapes, even when directories change concurrently.
type ChatStorage struct{ root *os.Root }

type ChatStoredFile struct {
	Key    string
	Size   int64
	SHA256 string
	MIME   string
}

func NewChatStorage(directory string) (*ChatStorage, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, fmt.Errorf("chat storage directory is required")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	if err = root.Chmod(".", 0700); err != nil {
		_ = root.Close()
		return nil, err
	}
	return &ChatStorage{root: root}, nil
}

func (s *ChatStorage) Close() error { return s.root.Close() }

func chatStorageKey(userID int64, id uuid.UUID) (string, error) {
	if userID <= 0 || id == uuid.Nil {
		return "", ErrChatNotFound
	}
	return path.Join("files", strconv.FormatInt(userID, 10), id.String()), nil
}

func chatImportKey(userID int64, importID, sessionID uuid.UUID) (string, error) {
	if userID <= 0 || importID == uuid.Nil || sessionID == uuid.Nil {
		return "", ErrChatNotFound
	}
	return path.Join("imports", strconv.FormatInt(userID, 10), importID.String(), sessionID.String()+".json"), nil
}

// WriteFile is called only after reserving this server-generated attachment ID
// in the database. Failed DB finalization keeps that reservation for the sweeper.
func (s *ChatStorage) WriteFile(ctx context.Context, userID int64, id uuid.UUID, source io.Reader, maxBytes int64) (*ChatStoredFile, error) {
	key, err := chatStorageKey(userID, id)
	if err != nil {
		return nil, err
	}
	if maxBytes <= 0 || maxBytes > ChatMaxFileBytes {
		return nil, ErrChatInvalid
	}
	return s.write(ctx, key, source, maxBytes)
}

func (s *ChatStorage) write(ctx context.Context, key string, source io.Reader, maxBytes int64) (*ChatStoredFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.root.Lstat(key); err == nil {
		return nil, os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := s.root.MkdirAll(path.Dir(key), 0700); err != nil {
		return nil, err
	}
	temporary := key + ".tmp"
	f, err := s.root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close(); _ = s.root.Remove(temporary) }()
	digest := sha256.New()
	head := make([]byte, 0, 512)
	buffer := make([]byte, 32*1024)
	reader := io.LimitReader(source, maxBytes+1)
	var size int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, readErr := reader.Read(buffer)
		if n > 0 {
			size += int64(n)
			if size > maxBytes {
				return nil, ErrChatTooLarge
			}
			if len(head) < 512 {
				head = append(head, buffer[:min(n, 512-len(head))]...)
			}
			if _, err := f.Write(buffer[:n]); err != nil {
				return nil, err
			}
			_, _ = digest.Write(buffer[:n])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	if err = f.Sync(); err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if err = s.root.Rename(temporary, key); err != nil {
		return nil, err
	}
	return &ChatStoredFile{Key: key, Size: size, SHA256: hex.EncodeToString(digest.Sum(nil)), MIME: http.DetectContentType(head)}, nil
}

func (s *ChatStorage) OpenFile(userID int64, id uuid.UUID) (*os.File, error) {
	key, err := chatStorageKey(userID, id)
	if err != nil {
		return nil, err
	}
	return s.root.Open(key)
}

func (s *ChatStorage) RemoveFile(userID int64, id uuid.UUID) error {
	key, err := chatStorageKey(userID, id)
	if err != nil {
		return err
	}
	for _, name := range []string{key, key + ".tmp"} {
		if err := s.root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s *ChatStorage) StageSession(ctx context.Context, userID int64, importID, sessionID uuid.UUID, source io.Reader) error {
	key, err := chatImportKey(userID, importID, sessionID)
	if err != nil {
		return err
	}
	_, err = s.write(ctx, key, source, ChatMaxSessionBytes)
	return err
}

func (s *ChatStorage) ReadStagedSession(userID int64, importID, sessionID uuid.UUID) ([]byte, error) {
	key, err := chatImportKey(userID, importID, sessionID)
	if err != nil {
		return nil, err
	}
	f, err := s.root.Open(key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	raw, err := io.ReadAll(io.LimitReader(f, ChatMaxSessionBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > ChatMaxSessionBytes {
		return nil, ErrChatTooLarge
	}
	return raw, nil
}

func (s *ChatStorage) RemoveImport(userID int64, importID uuid.UUID) error {
	if userID <= 0 || importID == uuid.Nil {
		return ErrChatNotFound
	}
	return s.root.RemoveAll(path.Join("imports", strconv.FormatInt(userID, 10), importID.String()))
}
