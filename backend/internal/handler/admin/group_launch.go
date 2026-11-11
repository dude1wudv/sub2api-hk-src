package admin

import (
	"encoding/json"
	"time"
)

// Distinguish omission (preserve) from null (unpublish) in PATCH-style updates.
type optionalLaunchTime struct {
	Set   bool
	Value *time.Time
}

func (v *optionalLaunchTime) UnmarshalJSON(data []byte) error {
	v.Set = true
	return json.Unmarshal(data, &v.Value)
}
