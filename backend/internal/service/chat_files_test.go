//go:build unit

package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func chatTestDocument(t *testing.T, extra ...string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range append([]string{"[Content_Types].xml", "word/document.xml"}, extra...) {
		file, err := writer.Create(name)
		require.NoError(t, err)
		text := "<xml>test</xml>"
		_, err = file.Write([]byte(text))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}
func TestChatFileTypesLimitsAndPrivatePreview(t *testing.T) {
	var imageBytes bytes.Buffer
	require.NoError(t, png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	for _, tc := range []struct {
		name string
		data []byte
		mime string
	}{
		{"image.png", imageBytes.Bytes(), "image/png"},
		{"document.pdf", []byte("%PDF-1.7 test"), "application/pdf"},
		{"document.docx", chatTestDocument(t), "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"notes.txt", []byte("你好，世界"), "text/plain; charset=utf-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ext, _, err := chatFileKind(tc.name)
			require.NoError(t, err)
			actual, err := chatValidateFile(bytes.NewReader(tc.data), int64(len(tc.data)), ext)
			require.NoError(t, err)
			require.Equal(t, tc.mime, actual)
		})
	}
	for _, name := range []string{"../private.txt", "C:" + string(rune(92)) + "private.txt", "a/b.txt", "a" + string(rune(92)) + "b.txt", "a" + string(rune(0)) + ".txt"} {
		_, _, err := chatFileKind(name)
		require.ErrorIs(t, err, ErrChatInvalid)
	}
	_, err := chatValidateFile(strings.NewReader("%PDF-1.7"), 8, "png")
	require.ErrorIs(t, err, ErrChatInvalid)
	_, err = chatValidateFile(bytes.NewReader([]byte{0, 0, 0, 1}), 4, "txt")
	require.ErrorIs(t, err, ErrChatInvalid)
	require.False(t, IsChatPreviewMIME("image/svg+xml"))
	require.False(t, IsChatPreviewMIME("text/html"))
	svc := &ChatService{}
	_, err = svc.UploadAttachment(context.Background(), 7, "large.txt", "", bytes.NewReader(nil), ChatMaxUserDataBytes+1, nil)
	require.ErrorIs(t, err, ErrChatTooLarge)
	_, err = svc.UploadAttachment(context.Background(), 7, "large.png", "", bytes.NewReader(nil), ChatMaxFileBytes+1, nil)
	require.ErrorIs(t, err, ErrChatTooLarge)
}
func TestChatDocumentZipRejectsTraversalDuplicatesAndBombs(t *testing.T) {
	for _, name := range []string{"../outside", "..", ".", "/absolute", "word/document.xml", "a" + string(rune(92)) + "b"} {
		t.Run(name, func(t *testing.T) {
			data := chatTestDocument(t, name)
			_, err := chatValidateFile(bytes.NewReader(data), int64(len(data)), "docx")
			require.ErrorIs(t, err, ErrChatInvalid)
		})
	}
	bomb := chatTestDocument(t)
	bombCentral := bytes.Index(bomb, []byte{0x50, 0x4b, 1, 2})
	require.GreaterOrEqual(t, bombCentral, 0)
	binary.LittleEndian.PutUint32(bomb[bombCentral+24:], (256<<20)+1)
	_, err := chatValidateFile(bytes.NewReader(bomb), int64(len(bomb)), "docx")
	require.ErrorIs(t, err, ErrChatTooLarge)
	corrupt := chatTestDocument(t)
	central := bytes.Index(corrupt, []byte{0x50, 0x4b, 1, 2})
	require.GreaterOrEqual(t, central, 0)
	corrupt[central+16] ^= 0xff
	_, err = chatValidateFile(bytes.NewReader(corrupt), int64(len(corrupt)), "docx")
	require.ErrorIs(t, err, ErrChatInvalid)
}
func TestChatImportPreservesBranchesButResetsAuthority(t *testing.T) {
	id := uuid.New()
	tree := chatTestTree(id)
	tree["presetId"] = uuid.NewString()
	tree["createdAt"] = int64(1000)
	tree["updatedAt"] = int64(2000)
	messages := tree["messages"].([]any)
	answer := messages[1].(map[string]any)
	answer["prefix"], answer["status"], answer["groupId"], answer["presetId"], answer["chatOptions"] = true, "streaming", 179, uuid.NewString(), map[string]any{"trusted": true}
	messages[0].(map[string]any)["files"] = []any{map[string]any{"id": uuid.NewString(), "name": "image.png", "size": 4, "serverId": "untrusted_remote"}}
	raw, err := json.Marshal(tree)
	require.NoError(t, err)
	raw = bytes.Replace(raw, []byte(`"role":"assistant"`), []byte(`"role":"`+string(rune(92))+`u0061ssistant"`), 1)
	imported, err := chatSanitizeImport(id, raw)
	require.NoError(t, err)
	require.Len(t, imported.Snapshot.Messages, 3)
	require.Equal(t, int64(1000), imported.CreatedAt.UnixMilli())
	require.Equal(t, int64(2000), imported.UpdatedAt.UnixMilli())
	require.Equal(t, "stopped", imported.Snapshot.Messages[1].Status)
	require.False(t, imported.Snapshot.Messages[1].Prefix)
	require.Zero(t, imported.Snapshot.Messages[1].GroupID)
	require.NotContains(t, string(imported.Snapshot.Raw), "untrusted_remote")
	require.NotContains(t, string(imported.Snapshot.Raw), "chatOptions")
	require.Contains(t, string(imported.Snapshot.Raw), "private-provider-signature")
	require.Contains(t, string(imported.Snapshot.Raw), "保留")
}
func TestChatImportManifestRejectsUnregisteredPathsAndSizes(t *testing.T) {
	id := uuid.New()
	item := ChatImportSession{SessionID: id, FileID: uuid.New(), Size: 1, SHA256: strings.Repeat("a", 64), Ready: true}
	manifest := ChatImportManifest{Sessions: map[string]ChatImportSession{id.String(): item}, Bytes: 1}
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	_, err = ParseChatImportManifest(raw)
	require.NoError(t, err)
	manifest.Bytes = 0
	raw, err = json.Marshal(manifest)
	require.NoError(t, err)
	_, err = ParseChatImportManifest(raw)
	require.ErrorIs(t, err, ErrChatInvalid)
	_, err = ParseChatImportManifest([]byte(`{"sessions":{"../../escape":{}},"bytes":0}`))
	require.ErrorIs(t, err, ErrChatInvalid)
}
