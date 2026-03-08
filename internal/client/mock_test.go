package client

import (
	"context"
	"errors"
	"testing"
)

func TestMockClient_DefaultResponse(t *testing.T) {
	m := NewMockClient()
	ctx := context.Background()

	resp, err := m.Do(ctx, &Request{Method: "GET", Path: "/v1/alerts"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if m.RequestCount() != 1 {
		t.Errorf("RequestCount = %d, want 1", m.RequestCount())
	}
	if m.LastRequest().Path != "/v1/alerts" {
		t.Errorf("LastRequest().Path = %q, want %q", m.LastRequest().Path, "/v1/alerts")
	}
}

func TestMockClient_QueuedResponses(t *testing.T) {
	m := NewMockClient()
	ctx := context.Background()

	m.RespondWith(200, map[string]string{"id": "alert-1"})
	m.RespondWith(404, map[string]string{"error": "not found"})

	resp1, err := m.Do(ctx, &Request{Method: "GET", Path: "/v1/alerts/1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp1.StatusCode != 200 {
		t.Errorf("first response StatusCode = %d, want 200", resp1.StatusCode)
	}

	resp2, err := m.Do(ctx, &Request{Method: "GET", Path: "/v1/alerts/2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.StatusCode != 404 {
		t.Errorf("second response StatusCode = %d, want 404", resp2.StatusCode)
	}

	// Third call falls back to default
	resp3, err := m.Do(ctx, &Request{Method: "GET", Path: "/v1/alerts/3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp3.StatusCode != 200 {
		t.Errorf("third response should fall back to default, StatusCode = %d", resp3.StatusCode)
	}
}

func TestMockClient_QueuedErrors(t *testing.T) {
	m := NewMockClient()
	ctx := context.Background()

	expectedErr := errors.New("connection refused")
	m.RespondWithError(expectedErr)

	_, err := m.Do(ctx, &Request{Method: "GET", Path: "/v1/alerts"})
	if !errors.Is(err, expectedErr) {
		t.Errorf("err = %v, want %v", err, expectedErr)
	}
}

func TestMockClient_DoFunc(t *testing.T) {
	m := NewMockClient()
	ctx := context.Background()

	m.DoFunc = func(_ context.Context, req *Request) (*Response, error) {
		if req.Path == "/v1/alerts" {
			return &Response{StatusCode: 200, Body: []byte(`{"alerts":[]}`)}, nil
		}
		return &Response{StatusCode: 404, Body: []byte(`{"error":"not found"}`)}, nil
	}

	resp, err := m.Do(ctx, &Request{Method: "GET", Path: "/v1/alerts"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	resp2, err := m.Do(ctx, &Request{Method: "GET", Path: "/v1/unknown"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp2.StatusCode)
	}
}

func TestMockClient_GetInstanceInfo(t *testing.T) {
	m := NewMockClient()

	info := m.GetInstanceInfo()
	if info.Region != "us-south" {
		t.Errorf("Region = %q, want %q", info.Region, "us-south")
	}
	if info.InstanceName != "test-instance" {
		t.Errorf("InstanceName = %q, want %q", info.InstanceName, "test-instance")
	}
}

func TestMockClient_Close(t *testing.T) {
	m := NewMockClient()

	if m.Closed {
		t.Error("should not be closed initially")
	}

	if err := m.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if !m.Closed {
		t.Error("should be closed after Close()")
	}
}

func TestMockClient_Reset(t *testing.T) {
	m := NewMockClient()
	ctx := context.Background()

	m.RespondWith(200, map[string]string{"id": "1"})
	_, _ = m.Do(ctx, &Request{Method: "GET", Path: "/test"})

	m.Reset()

	if m.RequestCount() != 0 {
		t.Errorf("RequestCount after Reset = %d, want 0", m.RequestCount())
	}
	if m.LastRequest() != nil {
		t.Error("LastRequest after Reset should be nil")
	}
}

func TestMockClient_DefaultError(t *testing.T) {
	m := NewMockClient()
	m.DefaultError = errors.New("always fail")
	m.DefaultResponse = nil
	ctx := context.Background()

	_, err := m.Do(ctx, &Request{Method: "GET", Path: "/test"})
	if err == nil {
		t.Error("expected default error")
	}
}
