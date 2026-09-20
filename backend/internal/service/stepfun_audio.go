package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
)

// StepFunAudioRequest is the validated OpenAI-compatible audio request that is
// forwarded unchanged to StepFun. StepFun supports JSON TTS and multipart ASR.
type StepFunAudioRequest struct {
	Model       string
	Endpoint    string
	ContentType string
	Body        []byte
	BodyHash    string
	AudioUsage  *AudioUsage
}

func (s *OpenAIGatewayService) ForwardStepFunAudio(ctx context.Context, c *gin.Context, account *Account, req StepFunAudioRequest) (*OpenAIForwardResult, error) {
	if account == nil || !account.IsStepFun() || account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("StepFun API-key account is required")
	}
	baseURL, err := s.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
	if err != nil {
		return nil, err
	}
	target := buildOpenAIEndpointURL(baseURL, req.Endpoint)
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	upstreamReq = upstreamReq.WithContext(WithHTTPUpstreamProfile(upstreamReq.Context(), HTTPUpstreamProfileOpenAI))
	headers, err := s.buildOpenAIAuthenticationHeaders(ctx, account, token)
	if err != nil {
		return nil, err
	}
	upstreamReq.Header = headers
	upstreamReq.Header.Set("Content-Type", req.ContentType)
	account.ApplyHeaderOverrides(upstreamReq.Header)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	started := time.Now()
	resp, err := s.doOpenAIUpstream(upstreamReq, proxyURL, account)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if readErr != nil {
			return nil, readErr
		}
		return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: body}
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Status(resp.StatusCode)
	c.Header("Content-Type", contentType)
	if req.Endpoint == "/v1/audio/transcriptions" {
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if readErr != nil {
			return nil, readErr
		}
		if _, writeErr := c.Writer.Write(responseBody); writeErr != nil {
			return nil, writeErr
		}
		// The transcription API does not report duration. Reuse the existing
		// conservative multipart-size estimator so configured hourly pricing is
		// actually applied instead of silently recording a free request.
		req.AudioUsage = estimateGrokVoiceAudioUsage("stt", req.Body, req.ContentType, responseBody, 0)
	} else if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		return nil, err
	}
	return &OpenAIForwardResult{
		RequestID: resp.Header.Get("x-request-id"), Model: req.Model, UpstreamModel: req.Model,
		UpstreamEndpoint: req.Endpoint, Duration: time.Since(started), AudioUsage: req.AudioUsage,
	}, nil
}
