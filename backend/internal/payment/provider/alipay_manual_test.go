//go:build unit

package provider

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestAlipayManualCreatePaymentReturnsStaticQRCode(t *testing.T) {
	t.Parallel()

	p := NewAlipayManual("manual-test")
	result, err := p.CreatePayment(context.Background(), payment.CreatePaymentRequest{Amount: "37.25"})

	require.NoError(t, err)
	require.Equal(t, "/alipay-manual-payment.jpg", result.QRCode)
	require.Equal(t, "CNY", result.Currency)
	require.Equal(t, payment.TypeAlipayManual, p.ProviderKey())
	require.Equal(t, []payment.PaymentType{payment.TypeAlipayManual}, p.SupportedTypes())
}
