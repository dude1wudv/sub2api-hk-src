package service

import (
	"encoding/json"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/passwordpolicy"
	"math"
)

type PasswordPolicy = passwordpolicy.Policy

const SettingKeyCardShopDenominationsUSD = "card_shop_denominations_usd"

func defaultCardShopDenominations() []float64 { return []float64{1, 10, 50} }
func normalizeCardShopDenominations(values []float64) ([]float64, error) {
	if values == nil {
		return defaultCardShopDenominations(), nil
	}
	if len(values) > 20 {
		return nil, infraerrors.BadRequest("CARD_DENOMINATIONS_INVALID", "at most 20 card denominations are allowed")
	}
	result := make([]float64, 0, len(values))
	seen := map[float64]bool{}
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return nil, infraerrors.BadRequest("CARD_DENOMINATIONS_INVALID", "card denominations must be positive USD amounts")
		}
		if !seen[v] {
			result = append(result, v)
			seen[v] = true
		}
	}
	return result, nil
}
func parseCardShopDenominations(raw string) []float64 {
	if raw == "" {
		return defaultCardShopDenominations()
	}
	var values []float64
	if json.Unmarshal([]byte(raw), &values) != nil {
		return defaultCardShopDenominations()
	}
	result, err := normalizeCardShopDenominations(values)
	if err != nil {
		return defaultCardShopDenominations()
	}
	return result
}
func (s *SettingService) heliosBootstrapURL() string {
	if s == nil || s.cfg == nil || !s.cfg.HeliosWorkbench.Enabled {
		return ""
	}
	return s.cfg.HeliosWorkbench.RedirectURI
}
