package service

import (
	"context"
	"testing"
	"time"
)

type summaryProxyRepoStub struct {
	ProxyRepository
	rows []Proxy
	err  error
}

func (s *summaryProxyRepoStub) ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

type summaryLatencyCacheStub struct {
	latencies map[int64]*ProxyLatencyInfo
	err       error
}

func (s *summaryLatencyCacheStub) GetProxyLatencies(ctx context.Context, proxyIDs []int64) (map[int64]*ProxyLatencyInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.latencies, nil
}

func (s *summaryLatencyCacheStub) SetProxyLatency(ctx context.Context, proxyID int64, info *ProxyLatencyInfo) error {
	return nil
}

func summaryProxyID(v int64) *int64 { return &v }

func TestAttachAccountSummaryProxyHealthSuccess(t *testing.T) {
	latency := int64(123)
	svc := &adminServiceImpl{
		proxyRepo: &summaryProxyRepoStub{rows: []Proxy{{ID: 1, Name: "Tokyo-1"}}},
		proxyLatencyCache: &summaryLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{
			1: {Success: true, LatencyMs: &latency, Message: "ok", UpdatedAt: time.Now()},
		}},
	}
	proxies := []AccountProxySummary{{ProxyID: summaryProxyID(1), Name: "Proxy #1", Total: 3, Available: 2}}
	svc.attachAccountSummaryProxyHealth(context.Background(), proxies)

	if proxies[0].Name != "Tokyo-1" {
		t.Fatalf("expected proxy name refreshed from repo, got %q", proxies[0].Name)
	}
	if proxies[0].LatencyStatus != "success" || proxies[0].LatencyMs == nil || *proxies[0].LatencyMs != 123 {
		t.Fatalf("unexpected latency fields: %#v", proxies[0])
	}
	if proxies[0].LatencyMessage != "ok" {
		t.Fatalf("unexpected latency message: %q", proxies[0].LatencyMessage)
	}
}

func TestAttachAccountSummaryProxyHealthFailedProbe(t *testing.T) {
	svc := &adminServiceImpl{
		proxyRepo: &summaryProxyRepoStub{rows: []Proxy{{ID: 2, Name: "Tokyo-2"}}},
		proxyLatencyCache: &summaryLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{
			2: {Success: false, Message: "connect timeout", UpdatedAt: time.Now()},
		}},
	}
	proxies := []AccountProxySummary{{ProxyID: summaryProxyID(2), Name: "Proxy #2", Total: 1, Available: 0}}
	svc.attachAccountSummaryProxyHealth(context.Background(), proxies)

	if proxies[0].LatencyStatus != "failed" {
		t.Fatalf("expected failed latency status, got %q", proxies[0].LatencyStatus)
	}
	if proxies[0].LatencyMs != nil {
		t.Fatalf("failed probe must not report latency_ms: %#v", proxies[0].LatencyMs)
	}
	if proxies[0].LatencyMessage != "connect timeout" {
		t.Fatalf("unexpected latency message: %q", proxies[0].LatencyMessage)
	}
}

func TestAttachAccountSummaryProxyHealthNoLatencySample(t *testing.T) {
	svc := &adminServiceImpl{
		proxyRepo:         &summaryProxyRepoStub{rows: []Proxy{{ID: 3, Name: "Tokyo-3"}}},
		proxyLatencyCache: &summaryLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{}},
	}
	proxies := []AccountProxySummary{{ProxyID: summaryProxyID(3), Name: "Proxy #3", Total: 2, Available: 0}}
	svc.attachAccountSummaryProxyHealth(context.Background(), proxies)

	if proxies[0].LatencyStatus != "" || proxies[0].LatencyMs != nil || proxies[0].LatencyMessage != "" {
		t.Fatalf("expected latency fields untouched without cache sample: %#v", proxies[0])
	}
	if proxies[0].Name != "Tokyo-3" {
		t.Fatalf("expected proxy name refreshed from repo, got %q", proxies[0].Name)
	}
}

func TestAttachAccountSummaryProxyHealthSkipsNilDependencies(t *testing.T) {
	svc := &adminServiceImpl{}
	proxies := []AccountProxySummary{
		{ProxyID: summaryProxyID(4), Name: "Proxy #4", Total: 1, Available: 1},
		{ProxyID: nil, Name: "No proxy", Total: 5, Available: 4},
	}
	// 无 proxyRepo / proxyLatencyCache 时应保持原样且不 panic。
	svc.attachAccountSummaryProxyHealth(context.Background(), proxies)

	if proxies[0].Name != "Proxy #4" || proxies[0].LatencyStatus != "" {
		t.Fatalf("expected untouched summary without dependencies: %#v", proxies[0])
	}
	if proxies[1].Name != "No proxy" {
		t.Fatalf("expected no-proxy bucket untouched: %#v", proxies[1])
	}
}
