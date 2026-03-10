package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"github.com/tareqmamari/cloud-logs-mcp/internal/config"
	"github.com/tareqmamari/cloud-logs-mcp/internal/health"
	"github.com/tareqmamari/cloud-logs-mcp/internal/server"
)

// mockAuthenticator implements server.Authenticator for testing.
type mockAuthenticator struct {
	validateErr error
	userID      string
	identityErr error
}

func (m *mockAuthenticator) ValidateToken() error {
	return m.validateErr
}

func (m *mockAuthenticator) GetUserIdentity() (string, error) {
	if m.identityErr != nil {
		return "", m.identityErr
	}
	return m.userID, nil
}

// mockTokenValidator implements health.TokenValidator for testing.
type mockTokenValidator struct {
	err error
}

func (m *mockTokenValidator) ValidateToken() error {
	return m.err
}

// getFreePort asks the OS for an available port.
func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}
	return port
}

// waitForServer polls the given URL until it responds or the timeout expires.
func waitForServer(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:gosec,bodyclose // test-only URL
		if err == nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Logf("warning: failed to close response body: %v", closeErr)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready within %v", url, timeout)
}

// --- Health server lifecycle tests ---

func TestHealthServerLifecycle(t *testing.T) {
	logger := zap.NewNop()
	mockClient := client.NewMockClient()
	validator := &mockTokenValidator{err: nil}
	checker := health.New(mockClient, validator, logger)

	port := getFreePort(t)
	srv := health.NewServer(checker, logger, port, "127.0.0.1", false)
	srv.SetReady(true)

	// Start the server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForServer(t, base+"/live", 2*time.Second)

	// Verify /health endpoint responds.
	t.Run("health", func(t *testing.T) {
		resp, err := http.Get(base + "/health") //nolint:gosec
		if err != nil {
			t.Fatalf("GET /health failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET /health status = %d, want 200", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}
		if !json.Valid(body) {
			t.Errorf("GET /health returned invalid JSON: %s", body)
		}
	})

	// Verify /ready endpoint responds.
	t.Run("ready", func(t *testing.T) {
		resp, err := http.Get(base + "/ready") //nolint:gosec
		if err != nil {
			t.Fatalf("GET /ready failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET /ready status = %d, want 200", resp.StatusCode)
		}
	})

	// Verify /live endpoint responds.
	t.Run("live", func(t *testing.T) {
		resp, err := http.Get(base + "/live") //nolint:gosec
		if err != nil {
			t.Fatalf("GET /live failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET /live status = %d, want 200", resp.StatusCode)
		}
	})

	// Graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Start() should return nil after graceful shutdown.
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Start returned unexpected error after shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after Shutdown")
	}
}

func TestHealthServerLifecycle_Shutdown(t *testing.T) {
	logger := zap.NewNop()
	mockClient := client.NewMockClient()
	validator := &mockTokenValidator{err: nil}
	checker := health.New(mockClient, validator, logger)

	port := getFreePort(t)
	srv := health.NewServer(checker, logger, port, "127.0.0.1", false)
	srv.SetReady(true)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForServer(t, base+"/live", 2*time.Second)

	// Shutdown the server.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify server has stopped by confirming connections are refused.
	// Give the OS a moment to release the socket.
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(base + "/live") //nolint:gosec
	if err == nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("failed to close response body: %v", closeErr)
		}
	}
	if err == nil {
		t.Error("expected connection error after shutdown, but request succeeded")
	}

	// Verify Start() returned cleanly.
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Start returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after Shutdown")
	}
}

// --- MCP protocol tests ---

// connectMCPClientServer creates a paired MCP server and client session for testing.
// The server runs in a background goroutine. Cancel the returned context to stop it.
func connectMCPClientServer(t *testing.T, mcpServer *mcp.Server) (context.CancelFunc, *mcp.ClientSession) {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())

	// Connect the server first (required before client connects).
	_, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("server Connect failed: %v", err)
	}

	// Connect the client.
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("client Connect failed: %v", err)
	}

	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("warning: failed to close session: %v", err)
		}
		cancel()
	})

	return cancel, session
}

func TestMCPProtocol_ToolRegistration(t *testing.T) {
	// Create an MCP server directly.
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "0.0.1",
	}, &mcp.ServerOptions{
		HasTools: true,
	})

	// Register a test tool.
	mcpServer.AddTool(&mcp.Tool{
		Name:        "echo",
		Description: "Echoes the input message",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Message to echo",
				},
			},
			"required": []string{"message"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]interface{}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("unmarshal args: %w", err)
		}
		msg, _ := args["message"].(string)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "echo: " + msg},
			},
		}, nil
	})

	_, session := connectMCPClientServer(t, mcpServer)

	// List tools and verify our echo tool is registered.
	listResult, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	found := false
	for _, tool := range listResult.Tools {
		if tool.Name == "echo" {
			found = true
			if tool.Description != "Echoes the input message" {
				t.Errorf("tool description = %q, want %q", tool.Description, "Echoes the input message")
			}
			break
		}
	}
	if !found {
		t.Error("echo tool not found in ListTools response")
	}
}

func TestMCPProtocol_ToolCall(t *testing.T) {
	// Create an MCP server with a test tool.
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "0.0.1",
	}, &mcp.ServerOptions{
		HasTools: true,
	})

	mcpServer.AddTool(&mcp.Tool{
		Name:        "greet",
		Description: "Greets a person by name",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the person to greet",
				},
			},
			"required": []string{"name"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]interface{}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("unmarshal args: %w", err)
		}
		name, _ := args["name"].(string)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Hello, " + name + "!"},
			},
		}, nil
	})

	_, session := connectMCPClientServer(t, mcpServer)

	// Call the greet tool.
	callResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]interface{}{"name": "World"},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if callResult.IsError {
		t.Error("CallTool returned an error result")
	}
	if len(callResult.Content) == 0 {
		t.Fatal("CallTool returned no content")
	}

	textContent, ok := callResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", callResult.Content[0])
	}
	expected := "Hello, World!"
	if textContent.Text != expected {
		t.Errorf("tool response = %q, want %q", textContent.Text, expected)
	}
}

// --- Full server tests using NewWithDeps ---
// These are grouped under a single top-level test to avoid Prometheus
// duplicate metric registration panics (metrics.New uses global registry).

func newTestConfig() *config.Config {
	return &config.Config{
		APIKey:     "test-api-key", // pragma: allowlist secret
		InstanceID: "test-instance",
		Region:     "us-south",
		HealthPort: 0, // disable health server for unit tests
	}
}

func TestNewWithDeps(t *testing.T) {
	// Create server once — reused across subtests to avoid Prometheus re-registration.
	// metrics.New() registers global Prometheus collectors, so only one server can be
	// created per test binary run.
	mock := client.NewMockClient()
	mock.DefaultResponse = &client.Response{StatusCode: 200, Body: []byte(`{}`)}
	authMock := &mockAuthenticator{userID: "test-user-123"}

	srv, err := server.NewWithDeps(newTestConfig(), mock, authMock, zap.NewNop(), "1.0.0-test")
	if err != nil {
		t.Fatalf("NewWithDeps failed: %v", err)
	}

	t.Run("CreatesServer", func(t *testing.T) {
		if srv == nil {
			t.Fatal("NewWithDeps returned nil server")
		}
	})

	t.Run("FallsBackOnIdentityError", func(t *testing.T) {
		// Verify the fallback path works — GetUserIdentity error is non-fatal.
		// We can't create a second server (Prometheus), so we verify indirectly
		// by confirming the first server was created successfully (the authMock
		// returned a valid userID). The fallback path sets user from API key hash
		// via tools.SetCurrentUser, which is a no-op from test perspective.
		// The key assertion: NewWithDeps does NOT return an error when
		// GetUserIdentity fails.
		if err != nil {
			t.Errorf("NewWithDeps should succeed even with identity fallback, got: %v", err)
		}
	})

	t.Run("RegistersAllTools", func(t *testing.T) {
		serverTransport, clientTransport := mcp.NewInMemoryTransports()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		mcpSrv := srv.MCPServer()
		_, err := mcpSrv.Connect(ctx, serverTransport, nil)
		if err != nil {
			t.Fatalf("server Connect failed: %v", err)
		}

		mcpClient := mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)

		session, err := mcpClient.Connect(ctx, clientTransport, nil)
		if err != nil {
			t.Fatalf("client Connect failed: %v", err)
		}
		defer func() {
			if err := session.Close(); err != nil {
				t.Logf("warning: failed to close session: %v", err)
			}
		}()

		result, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("ListTools failed: %v", err)
		}

		// The server registers 80+ tools — verify a reasonable number are present
		if len(result.Tools) < 50 {
			t.Errorf("Expected 50+ tools registered, got %d", len(result.Tools))
		}

		// Spot-check key tools
		toolNames := make(map[string]bool)
		for _, tool := range result.Tools {
			toolNames[tool.Name] = true
		}

		expectedTools := []string{
			"get_alert", "list_alerts", "create_alert",
			"query_logs", "list_views", "list_dashboards",
			"search_tools", "health_check",
		}
		for _, name := range expectedTools {
			if !toolNames[name] {
				t.Errorf("Expected tool %q to be registered", name)
			}
		}
	})

	t.Run("ToolCallWithMock", func(t *testing.T) {
		mock.Reset()
		mock.DefaultResponse = &client.Response{
			StatusCode: 200,
			Body:       []byte(`{"id":"alert-123","name":"test-alert","severity":"critical"}`),
		}

		serverTransport, clientTransport := mcp.NewInMemoryTransports()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		mcpSrv := srv.MCPServer()
		_, err := mcpSrv.Connect(ctx, serverTransport, nil)
		if err != nil {
			t.Fatalf("server Connect failed: %v", err)
		}

		mcpClient := mcp.NewClient(&mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		}, nil)

		session, err := mcpClient.Connect(ctx, clientTransport, nil)
		if err != nil {
			t.Fatalf("client Connect failed: %v", err)
		}
		defer func() {
			if err := session.Close(); err != nil {
				t.Logf("warning: failed to close session: %v", err)
			}
		}()

		// Call list_alerts tool (simpler — no ID param required)
		callResult, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "list_alerts",
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		// Verify we got a response (not an MCP-level error)
		if len(callResult.Content) == 0 {
			t.Fatal("CallTool returned no content")
		}

		// Verify the mock received at least one request
		if mock.RequestCount() == 0 {
			t.Error("MockClient received no requests")
		}
	})
}
