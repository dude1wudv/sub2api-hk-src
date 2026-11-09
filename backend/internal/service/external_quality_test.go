package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func externalQualityFixture(now time.Time) ExternalQualityImport {
	input := ExternalQualityImport{Source: "mdkj", GroupID: 72, RemoteGroupID: 62}
	code := 0
	input.Payload.Code = &code
	input.Payload.Data = []ExternalQualityRemoteEvent{{ID: "a78777d8-c87c-437a-b615-9a2ec1374055", GroupID: 62, StartedAt: now.Add(-time.Minute), FinishedAt: now, Model: "gpt-6-astra", ResponseModel: "gpt-6-astra", Status: "completed", HTML: "<!doctype html><h1>test</h1>", DurationMs: 60000}}
	return input
}

func TestExternalQualityImportMappingAndValidation(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		change func(*ExternalQualityImport)
	}{
		{"wrong mapping", func(i *ExternalQualityImport) { i.RemoteGroupID = 170 }},
		{"deepseek", func(i *ExternalQualityImport) { i.GroupID = 179 }},
		{"wrong event group", func(i *ExternalQualityImport) { i.Payload.Data[0].GroupID = 170 }},
		{"unsupported source", func(i *ExternalQualityImport) { i.Source = "other" }},
		{"fabricated verdict", func(i *ExternalQualityImport) { i.Payload.Data[0].Status = "success" }},
		{"wrong model", func(i *ExternalQualityImport) { i.Payload.Data[0].Model = "deepseek-v4.1-flash" }},
		{"invalid uuid", func(i *ExternalQualityImport) { i.Payload.Data[0].ID = "not-an-event" }},
		{"future", func(i *ExternalQualityImport) { i.Payload.Data[0].FinishedAt = now.Add(6 * time.Minute) }},
		{"backwards", func(i *ExternalQualityImport) { i.Payload.Data[0].FinishedAt = now.Add(-time.Hour) }},
		{"oversized artwork", func(i *ExternalQualityImport) { i.Payload.Data[0].HTML = strings.Repeat("x", (256<<10)+1) }},
		{"duplicate", func(i *ExternalQualityImport) { i.Payload.Data = append(i.Payload.Data, i.Payload.Data[0]) }},
		{"missing code", func(i *ExternalQualityImport) { i.Payload.Code = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := externalQualityFixture(now)
			test.change(&input)
			_, err := normalizeExternalQualityImport(input, 42, now)
			require.Error(t, err)
		})
	}
	for _, pair := range [][2]int64{{72, 62}, {237, 170}} {
		input := externalQualityFixture(now)
		input.GroupID = pair[0]
		input.RemoteGroupID = pair[1]
		input.Payload.Data[0].GroupID = pair[1]
		records, err := normalizeExternalQualityImport(input, 42, now)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Len(t, records[0].ContentSHA256, 64)
		require.Equal(t, int64(42), records[0].ImportedBy)
	}
}

type externalQualityRepoStub struct {
	ChannelMonitorV2Repository
	events []ExternalQualityEvent
}

func (r *externalQualityRepoStub) ImportExternalQuality(context.Context, []ExternalQualityRecord) (int, error) {
	return 1, nil
}
func (r *externalQualityRepoStub) ListExternalQuality(context.Context, int64) ([]ExternalQualityEvent, error) {
	return r.events, nil
}
func (r *externalQualityRepoStub) GetExternalQualityArtwork(context.Context, int64, int64) (string, error) {
	return "", ErrExternalQualityNotFound
}

func TestExternalQualityReportUsesOriginalTestTimeAndNoVerdict(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	repo := &externalQualityRepoStub{events: []ExternalQualityEvent{{ID: 123, ModelID: "gpt-6-astra", Status: "completed", CreatedAt: now.Add(-4 * time.Hour), FinishedAt: now.Add(-3 * time.Hour), ImportedAt: now, HasArtwork: true}}}
	svc := NewChannelMonitorV2Service(repo)
	svc.now = func() time.Time { return now }
	report, err := svc.ExternalQuality(context.Background(), 72)
	require.NoError(t, err)
	require.Equal(t, "stale", report.Status)
	require.Equal(t, now.Add(-3*time.Hour), *report.LastSuccessAt)
	raw, err := json.Marshal(report)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"score":null`)
	for _, forbidden := range []string{"source", "mdkj", "remote_group", "imported_at", "success\""} {
		require.NotContains(t, string(raw), forbidden)
	}
	repo.events[0].FinishedAt = now.Add(-time.Hour)
	report, err = svc.ExternalQuality(context.Background(), 72)
	require.NoError(t, err)
	require.Equal(t, "ready", report.Status)
	repo.events = nil
	report, err = svc.ExternalQuality(context.Background(), 237)
	require.NoError(t, err)
	require.Equal(t, "unavailable", report.Status)
	require.NotNil(t, report.Events)
	_, err = svc.ExternalQuality(context.Background(), 179)
	require.ErrorIs(t, err, ErrExternalQualityNotFound)
}
