package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
)

const ChatAuxiliaryPriceBasis = "deepseek-cny-list-1to1-v1"

type ChatAuxiliaryUsage struct {
	PriceBasis string                   `json:"price_basis"`
	Provider   string                   `json:"provider"`
	Items      []ChatAuxiliaryUsageItem `json:"items"`
}

type ChatAuxiliaryUsageItem struct {
	Kind           string `json:"kind"`
	Quantity       *int64 `json:"quantity"`
	Unit           string `json:"unit"`
	UnitPrice      string `json:"unit_price"`
	PriceQuantity  int64  `json:"price_quantity"`
	ChargedCost    string `json:"charged_cost"`
	MeteringStatus string `json:"metering_status"`
	QuantitySource string `json:"quantity_source"`
}

type ChatAuxiliaryPrice struct {
	Kind          string `json:"kind"`
	Unit          string `json:"unit"`
	UnitPrice     string `json:"unit_price"`
	PriceQuantity int64  `json:"price_quantity"`
}

func chatAuxiliaryPrices() []ChatAuxiliaryPrice {
	prices := make([]ChatAuxiliaryPrice, 0, 6)
	for _, kind := range []string{"search", "fetch", "image_search", "ocr", "asr", "tts"} {
		unit, price, quantity, _ := chatAuxiliaryPrice(kind)
		prices = append(prices, ChatAuxiliaryPrice{Kind: kind, Unit: unit, UnitPrice: price, PriceQuantity: quantity})
	}
	return prices
}

func chatAuxiliaryPrice(kind string) (unit, price string, quantity int64, ok bool) {
	switch kind {
	case "search", "image_search", "fetch":
		return "call", "0.005", 1, true
	case "ocr":
		return "page", "0.0015", 1, true
	case "asr":
		return "second", "0.78", 3600, true
	case "tts":
		return "character", "3", 10000, true
	case "native_files":
		return "page", "0", 1, true
	}
	return "", "", 0, false
}

func chatAuxiliaryUsage(kind string, quantity *int64, status string) (*ChatAuxiliaryUsage, error) {
	unit, price, priceQuantity, ok := chatAuxiliaryPrice(kind)
	if !ok || (quantity != nil && *quantity < 0) || (status != "confirmed" && status != "unreported" && status != "incomplete") || (status == "confirmed" && quantity == nil) {
		return nil, errors.New("invalid chat auxiliary metering")
	}
	source := "provider_usage"
	if kind == "tts" {
		source = "request_text"
	}
	if quantity == nil {
		source = "provider_unreported"
	}
	amount := decimal.Zero
	if status == "confirmed" && quantity != nil {
		amount = decimal.NewFromInt(*quantity).Mul(decimal.RequireFromString(price)).Div(decimal.NewFromInt(priceQuantity))
	}
	return &ChatAuxiliaryUsage{PriceBasis: ChatAuxiliaryPriceBasis, Provider: "deepseek", Items: []ChatAuxiliaryUsageItem{{Kind: kind, Quantity: quantity, Unit: unit, UnitPrice: price, PriceQuantity: priceQuantity, ChargedCost: amount.Round(UsageBillingMonetaryScale).StringFixed(UsageBillingMonetaryScale), MeteringStatus: status, QuantitySource: source}}}, nil
}

func (u *ChatAuxiliaryUsage) cost() (float64, error) {
	if u == nil {
		return 0, nil
	}
	if u.PriceBasis != ChatAuxiliaryPriceBasis || u.Provider != "deepseek" || len(u.Items) == 0 || len(u.Items) > 32 {
		return 0, errors.New("invalid chat auxiliary price basis")
	}
	total := decimal.Zero
	for _, item := range u.Items {
		canonical, err := chatAuxiliaryUsage(item.Kind, item.Quantity, item.MeteringStatus)
		if err != nil {
			return 0, err
		}
		expected := canonical.Items[0]
		if item.Unit != expected.Unit || item.UnitPrice != expected.UnitPrice || item.PriceQuantity != expected.PriceQuantity || item.QuantitySource != expected.QuantitySource || item.ChargedCost != expected.ChargedCost {
			return 0, errors.New("invalid chat auxiliary price snapshot")
		}
		if item.MeteringStatus == "confirmed" {
			total = total.Add(decimal.NewFromInt(*item.Quantity).Mul(decimal.RequireFromString(item.UnitPrice)).Div(decimal.NewFromInt(item.PriceQuantity)))
		}
	}
	value, _ := total.Float64()
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, errors.New("invalid chat auxiliary amount")
	}
	return QuantizeUsageBillingAmount(value), nil
}

func ChatAuxiliaryTTSCharacters(text string) (int64, error) {
	if !utf8.ValidString(text) {
		return 0, ErrChatInvalid
	}
	var count int64
	for _, r := range text {
		count++
		if unicode.Is(unicode.Han, r) {
			count++
		}
	}
	return count, nil
}

func chatAuxiliaryASRSeconds(durationMS int64) (int64, error) {
	if durationMS <= 0 || durationMS > 120000 {
		return 0, ErrChatInvalid
	}
	return (durationMS + 999) / 1000, nil
}

func chatNativeAuxiliaryUsage(ctx context.Context, account *Account, raw []byte) *ChatAuxiliaryUsage {
	native, _ := ctx.Value(chatNativeContextKey{}).(*chatNativeRequest)
	if native == nil || !chatNativeDeepSeekAccount(account) || (!native.extensions && len(native.files) == 0) {
		return nil
	}
	usage := &ChatAuxiliaryUsage{PriceBasis: ChatAuxiliaryPriceBasis, Provider: "deepseek"}
	if native.extensions {
		for _, meter := range []struct{ kind, field string }{{"search", "tool_search"}, {"fetch", "tool_fetch"}, {"image_search", "tool_image_search"}} {
			var quantity *int64
			status := "unreported"
			value := gjson.GetBytes(raw, "usage.server_tool_use."+meter.field)
			if value.Type == gjson.Number {
				if count, err := strconv.ParseInt(value.Raw, 10, 64); err == nil && count >= 0 {
					quantity = &count
					status = "confirmed"
				}
			}
			item, _ := chatAuxiliaryUsage(meter.kind, quantity, status)
			usage.Items = append(usage.Items, item.Items...)
		}
	}
	if len(native.files) > 0 {
		item, _ := chatAuxiliaryUsage("native_files", nil, "unreported")
		usage.Items = append(usage.Items, item.Items...)
	}
	return usage
}

func chatNativeAuxiliaryTerminalUsage(raw []byte) bool {
	if !gjson.GetBytes(raw, "usage.server_tool_use").IsObject() {
		return false
	}
	switch gjson.GetBytes(raw, "type").String() {
	case "message_delta":
		return gjson.GetBytes(raw, "delta.stop_reason").String() != ""
	case "error", "message_stop", "message", "":
		return true
	}
	return false
}

// ChatNativeAuxiliaryReservationCost is not a final fee or a tool-call limit.
func ChatNativeAuxiliaryReservationCost(ctx context.Context) float64 {
	native, _ := ctx.Value(chatNativeContextKey{}).(*chatNativeRequest)
	if native != nil && native.extensions {
		return 0.10
	}
	return 0
}
