// Package client provides HTTP client functionality for IBM Cloud Logs API.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// MockClient implements Doer for testing. It records all requests and returns
// configurable responses, enabling unit tests for components that depend on
// the API client without requiring real IBM Cloud credentials.
type MockClient struct {
	mu sync.Mutex

	// DoFunc is called for each Do() invocation. If nil, the default
	// behavior uses Responses/Errors slices or returns a 200 empty response.
	DoFunc func(ctx context.Context, req *Request) (*Response, error)

	// Responses is a queue of responses to return (FIFO). Each call to Do()
	// pops the first entry. When empty, falls back to DefaultResponse.
	Responses []*Response

	// Errors is a queue of errors to return (FIFO), paired with Responses.
	// If Errors[i] is non-nil, it is returned instead of Responses[i].
	Errors []error

	// DefaultResponse is returned when Responses is empty and DoFunc is nil.
	DefaultResponse *Response

	// DefaultError is returned when Errors is empty, Responses is empty, and DoFunc is nil.
	DefaultError error

	// Requests records all requests received by Do(), in order.
	Requests []*Request

	// InstanceInfo is returned by GetInstanceInfo().
	Instance InstanceInfo

	// Closed tracks whether Close() was called.
	Closed bool
}

// Verify MockClient implements Doer at compile time.
var _ Doer = (*MockClient)(nil)

// NewMockClient creates a MockClient with sensible defaults.
func NewMockClient() *MockClient {
	return &MockClient{
		DefaultResponse: &Response{
			StatusCode: 200,
			Body:       []byte(`{}`),
		},
		Instance: InstanceInfo{
			ServiceURL:   "https://test.api.us-south.logs.cloud.ibm.com",
			Region:       "us-south",
			InstanceName: "test-instance",
		},
	}
}

// Do executes a mock API request. It records the request and returns the
// configured response.
func (m *MockClient) Do(ctx context.Context, req *Request) (*Response, error) {
	m.mu.Lock()
	m.Requests = append(m.Requests, req)

	// Custom handler takes priority
	if m.DoFunc != nil {
		m.mu.Unlock()
		return m.DoFunc(ctx, req)
	}

	// Pop from queues
	if len(m.Responses) > 0 || len(m.Errors) > 0 {
		var resp *Response
		var err error

		if len(m.Responses) > 0 {
			resp = m.Responses[0]
			m.Responses = m.Responses[1:]
		}
		if len(m.Errors) > 0 {
			err = m.Errors[0]
			m.Errors = m.Errors[1:]
		}

		m.mu.Unlock()
		if err != nil {
			return nil, err
		}
		if resp != nil {
			return resp, nil
		}
	} else {
		m.mu.Unlock()
	}

	// Defaults
	if m.DefaultError != nil {
		return nil, m.DefaultError
	}
	return m.DefaultResponse, nil
}

// GetInstanceInfo returns the configured instance info.
func (m *MockClient) GetInstanceInfo() InstanceInfo {
	return m.Instance
}

// Close marks the client as closed.
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Closed = true
	return nil
}

// Reset clears all recorded requests and queued responses.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Requests = nil
	m.Responses = nil
	m.Errors = nil
	m.Closed = false
}

// LastRequest returns the most recent request, or nil if none recorded.
func (m *MockClient) LastRequest() *Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Requests) == 0 {
		return nil
	}
	return m.Requests[len(m.Requests)-1]
}

// RequestCount returns the number of requests recorded.
func (m *MockClient) RequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Requests)
}

// RespondWith queues a single JSON response body with the given status code.
func (m *MockClient) RespondWith(statusCode int, body interface{}) {
	data, err := json.Marshal(body)
	if err != nil {
		panic(fmt.Sprintf("MockClient.RespondWith: failed to marshal body: %v", err))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses = append(m.Responses, &Response{
		StatusCode: statusCode,
		Body:       data,
	})
	m.Errors = append(m.Errors, nil)
}

// RespondWithError queues an error response.
func (m *MockClient) RespondWithError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses = append(m.Responses, nil)
	m.Errors = append(m.Errors, err)
}
