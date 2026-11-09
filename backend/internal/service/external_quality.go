package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

var ErrExternalQualityNotFound = infraerrors.NotFound("EXTERNAL_QUALITY_NOT_FOUND", "Test result not found")

const ExternalQualityMaxImportBytes int64 = 4 << 20

// ExternalQualityEvent deliberately contains no foreign source/account/event IDs.
// completed means artwork generation finished, never a passing quality verdict.
type ExternalQualityEvent struct {
	ID            int64     `json:"id"`
	ModelID       string    `json:"model_id"`
	ResponseModel string    `json:"response_model"`
	Status        string    `json:"status"`
	Score         *float64  `json:"score"`
	CreatedAt     time.Time `json:"created_at"`
	FinishedAt    time.Time `json:"finished_at"`
	DurationMs    int64     `json:"duration_ms"`
	HasArtwork    bool      `json:"has_artwork"`
	ImportedAt    time.Time `json:"-"`
}

type ExternalQualityReport struct {
	GroupID       int64                  `json:"group_id"`
	Model         string                 `json:"model"`
	UpdatedAt     *time.Time             `json:"updated_at"`
	LastSuccessAt *time.Time             `json:"last_success_at"`
	Status        string                 `json:"status"`
	Events        []ExternalQualityEvent `json:"events"`
}

type ExternalQualityRemoteEvent struct {
	ID            string    `json:"id"`
	GroupID       int64     `json:"group_id"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	Status        string    `json:"status"`
	Model         string    `json:"model"`
	ResponseModel string    `json:"response_model"`
	HTML          string    `json:"html"`
	DurationMs    int64     `json:"duration_ms"`
}

type ExternalQualityImport struct {
	Source        string `json:"source"`
	GroupID       int64  `json:"group_id"`
	RemoteGroupID int64  `json:"remote_group_id"`
	Payload       struct {
		Code *int                         `json:"code"`
		Data []ExternalQualityRemoteEvent `json:"data"`
	} `json:"payload"`
}

// ExternalQualityRecord is a persistence-only structure; do not return it to users.
type ExternalQualityRecord struct {
	Source        string
	SourceEventID string
	RemoteGroupID int64
	GroupID       int64
	Event         ExternalQualityRemoteEvent
	ContentSHA256 string
	ImportedBy    int64
}

type ExternalQualityImportResult struct {
	GroupID  int64 `json:"group_id"`
	Imported int   `json:"imported"`
	Existing int   `json:"existing"`
}

// This optional port keeps the passive-monitor mock contract independent of
// externally imported artifacts. The production repository implements both.
type ExternalQualityRepository interface {
	ImportExternalQuality(context.Context, []ExternalQualityRecord) (int, error)
	ListExternalQuality(context.Context, int64) ([]ExternalQualityEvent, error)
	GetExternalQualityArtwork(context.Context, int64, int64) (string, error)
}

func ExternalQualityGroupEnabled(groupID int64) bool { return groupID == 72 || groupID == 237 }

func externalQualityInvalid(message string) error {
	return infraerrors.BadRequest("INVALID_EXTERNAL_QUALITY", message)
}

func (s *ChannelMonitorV2Service) ImportExternalQuality(ctx context.Context, input ExternalQualityImport, userID int64) (*ExternalQualityImportResult, error) {
	if s == nil {
		return nil, errors.New("external quality service unavailable")
	}
	records, err := normalizeExternalQualityImport(input, userID, s.now())
	if err != nil {
		return nil, err
	}
	repo, ok := s.repo.(ExternalQualityRepository)
	if !ok {
		return nil, errors.New("external quality persistence unavailable")
	}
	count, err := repo.ImportExternalQuality(ctx, records)
	if err != nil {
		return nil, err
	}
	return &ExternalQualityImportResult{GroupID: input.GroupID, Imported: count, Existing: len(records) - count}, nil
}

func normalizeExternalQualityImport(input ExternalQualityImport, userID int64, now time.Time) ([]ExternalQualityRecord, error) {
	if userID <= 0 || input.Source != "mdkj" || !ExternalQualityGroupEnabled(input.GroupID) ||
		(input.GroupID == 72 && input.RemoteGroupID != 62) || (input.GroupID == 237 && input.RemoteGroupID != 170) {
		return nil, externalQualityInvalid("Unsupported source or group mapping")
	}
	if input.Payload.Code == nil || *input.Payload.Code != 0 || len(input.Payload.Data) == 0 || len(input.Payload.Data) > 40 {
		return nil, externalQualityInvalid("Expected a successful history response with 1..40 records")
	}
	seen := make(map[string]bool, len(input.Payload.Data))
	records := make([]ExternalQualityRecord, 0, len(input.Payload.Data))
	for i, event := range input.Payload.Data {
		id, err := uuid.Parse(event.ID)
		if err != nil || id == uuid.Nil || seen[id.String()] {
			return nil, externalQualityInvalid(fmt.Sprintf("Record %d has an invalid or repeated event ID", i+1))
		}
		seen[id.String()] = true
		event.ID = id.String()
		if event.GroupID != input.RemoteGroupID || event.Status != "completed" || (event.Model != "gpt-6-astra" && event.Model != "gpt-6.1-sol") {
			return nil, externalQualityInvalid(fmt.Sprintf("Record %d has an unexpected group, model or status", i+1))
		}
		if len(event.ResponseModel) > 128 || !utf8.ValidString(event.ResponseModel) || strings.ContainsAny(event.ResponseModel, "\x00\r\n") {
			return nil, externalQualityInvalid("Invalid response model")
		}
		if event.StartedAt.Before(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) || event.StartedAt.After(now.Add(5*time.Minute)) || event.FinishedAt.Before(event.StartedAt) || event.FinishedAt.After(now.Add(5*time.Minute)) || event.FinishedAt.Sub(event.StartedAt) > 24*time.Hour || event.DurationMs < 0 || event.DurationMs > int64((24*time.Hour)/time.Millisecond) {
			return nil, externalQualityInvalid("Invalid test timestamps or duration")
		}
		if len(event.HTML) > 256<<10 || !utf8.ValidString(event.HTML) || strings.ContainsRune(event.HTML, 0) {
			return nil, externalQualityInvalid("Invalid or oversized test artwork")
		}
		// Only allowlisted fields are persisted. Never preserve arbitrary envelope
		// fields, headers or credentials from a copied browser response.
		raw, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(raw)
		records = append(records, ExternalQualityRecord{Source: input.Source, SourceEventID: event.ID, GroupID: input.GroupID, RemoteGroupID: input.RemoteGroupID, Event: event, ContentSHA256: hex.EncodeToString(digest[:]), ImportedBy: userID})
	}
	return records, nil
}

func (s *ChannelMonitorV2Service) ExternalQuality(ctx context.Context, groupID int64) (*ExternalQualityReport, error) {
	if !ExternalQualityGroupEnabled(groupID) {
		return nil, ErrExternalQualityNotFound
	}
	report := &ExternalQualityReport{GroupID: groupID, Status: "unavailable", Events: []ExternalQualityEvent{}}
	if s == nil {
		return nil, errors.New("external quality service unavailable")
	}
	repo, ok := s.repo.(ExternalQualityRepository)
	if !ok {
		return nil, errors.New("external quality persistence unavailable")
	}
	events, err := repo.ListExternalQuality(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return report, nil
	}
	report.Events = events
	latest := events[0].FinishedAt
	updated := events[0].ImportedAt
	for _, event := range events {
		if event.FinishedAt.After(latest) {
			latest = event.FinishedAt
		}
		if event.ImportedAt.After(updated) {
			updated = event.ImportedAt
		}
	}
	report.UpdatedAt, report.LastSuccessAt = &updated, &latest
	report.Status = "ready"
	if s.now().Sub(latest) > 2*time.Hour {
		report.Status = "stale"
	}
	return report, nil
}

func (s *ChannelMonitorV2Service) ExternalQualityArtwork(ctx context.Context, groupID, id int64) (string, error) {
	if s == nil || id <= 0 || !ExternalQualityGroupEnabled(groupID) {
		return "", ErrExternalQualityNotFound
	}
	repo, ok := s.repo.(ExternalQualityRepository)
	if !ok {
		return "", errors.New("external quality persistence unavailable")
	}
	return repo.GetExternalQualityArtwork(ctx, groupID, id)
}
