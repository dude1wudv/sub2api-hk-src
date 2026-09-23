package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMirasimAccountTestRequiresVisibleOutputAndCompletion(t *testing.T) {
	tests := []struct {
		name      string
		stream    string
		wantText  string
		wantError bool
	}{
		{
			name: "claude text",
			stream: "event: message_start\ndata: {\"type\":\"message_start\"}\n\n" +
				"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"hello\"}}\n\n" +
				"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
			wantText: "hello",
		},
		{
			name:      "first handshake frame only",
			stream:    "event: message_start\ndata: {\"type\":\"message_start\"}\n\n",
			wantError: true,
		},
		{
			name:      "delayed relay error",
			stream:    "data: {\"type\":\"message_start\"}\n\ndata: {\"type\":\"error\",\"error\":{\"message\":\"quota exceeded\"}}\n\n",
			wantError: true,
		},
		{
			name:      "empty completion",
			stream:    "data: {\"type\":\"message_stop\"}\n\n",
			wantError: true,
		},
		{
			name:     "chat text",
			stream:   "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: [DONE]\n\n",
			wantText: "hello",
		},
		{
			name:     "responses text",
			stream:   "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\ndata: {\"type\":\"response.completed\"}\n\n",
			wantText: "hello",
		},
		{
			name:      "text without terminal frame",
			stream:    "data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"hello\"}}\n\n",
			wantText:  "hello",
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
			svc := &AccountTestService{}
			err := svc.processMirasimTestStream(ctx, strings.NewReader(tt.stream))
			if (err != nil) != tt.wantError {
				t.Fatalf("error=%v, wantError=%v", err, tt.wantError)
			}
			output := recorder.Body.String()
			if tt.wantText != "" && !strings.Contains(output, `"text":"`+tt.wantText+`"`) {
				t.Fatalf("missing model output %q in %s", tt.wantText, output)
			}
			if strings.Contains(output, `"type":"test_complete"`) == tt.wantError {
				t.Fatalf("incorrect completion event: %s", output)
			}
		})
	}
}

func TestMirasimAccountTestDisplaysNonStreamingJSON(t *testing.T) {
	for _, body := range []string{
		`{"choices":[{"message":{"content":"hello"}}]}`,
		`{"content":[{"type":"text","text":"hello"}]}`,
		`{"output":[{"content":[{"type":"output_text","text":"hello"}]}]}`,
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
		svc := &AccountTestService{}
		if err := svc.processMirasimTestJSON(ctx, strings.NewReader(body)); err != nil {
			t.Fatalf("valid JSON response failed: %v", err)
		}
		if !strings.Contains(recorder.Body.String(), `"text":"hello"`) {
			t.Fatalf("missing text output: %s", recorder.Body.String())
		}
	}
}
