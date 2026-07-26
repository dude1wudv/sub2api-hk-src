package provider

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// AlipayManual is a static QR payment method that requires administrator confirmation.
type AlipayManual struct{ instanceID string }

func NewAlipayManual(instanceID string) *AlipayManual { return &AlipayManual{instanceID: instanceID} }
func (p *AlipayManual) Name() string                  { return "Alipay manual" }
func (p *AlipayManual) ProviderKey() string           { return payment.TypeAlipayManual }
func (p *AlipayManual) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeAlipayManual}
}
func (p *AlipayManual) CreatePayment(context.Context, payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	return &payment.CreatePaymentResponse{QRCode: "/alipay-manual-payment.jpg", Currency: "CNY"}, nil
}
func (p *AlipayManual) QueryOrder(context.Context, string) (*payment.QueryOrderResponse, error) {
	return &payment.QueryOrderResponse{Status: payment.ProviderStatusPending}, nil
}
func (p *AlipayManual) VerifyNotification(context.Context, string, map[string]string) (*payment.PaymentNotification, error) {
	return nil, nil
}
func (p *AlipayManual) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return &payment.RefundResponse{Status: payment.ProviderStatusFailed}, nil
}
