//go:build unit

package admin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupLaunchUpdateJSONPresence(t *testing.T) {
	for _, tc := range []struct {
		body                    string
		set, hasTime, wantError bool
	}{
		{`{}`, false, false, false},
		{`{"launched_at":null}`, true, false, false},
		{`{"launched_at":"2026-11-11T12:00:00Z"}`, true, true, false},
		{`{"launched_at":"invalid"}`, true, false, true},
	} {
		t.Run(tc.body, func(t *testing.T) {
			var req UpdateGroupRequest
			err := json.Unmarshal([]byte(tc.body), &req)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.set, req.LaunchedAt.Set)
			require.Equal(t, tc.hasTime, req.LaunchedAt.Value != nil)
		})
	}
}
