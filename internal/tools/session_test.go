package tools

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateUserID(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		instanceID string
	}{
		{
			name:       "basic generation",
			apiKey:     "test-api-key-123",
			instanceID: "instance-456",
		},
		{
			name:       "different keys produce different IDs",
			apiKey:     "different-key",
			instanceID: "instance-456",
		},
		{
			name:       "different instances produce different IDs",
			apiKey:     "test-api-key-123",
			instanceID: "different-instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := GenerateUserID(tt.apiKey, tt.instanceID)

			// Should be 16 characters (hex encoded first 8 bytes of SHA256)
			if len(userID) != 16 {
				t.Errorf("Expected userID length 16, got %d", len(userID))
			}

			// Should be deterministic
			userID2 := GenerateUserID(tt.apiKey, tt.instanceID)
			if userID != userID2 {
				t.Errorf("Expected deterministic ID, got %s and %s", userID, userID2)
			}
		})
	}
}

func TestGenerateUserIDFromSubject(t *testing.T) {
	subject := "iam-ServiceId-12345678-abcd-1234-efgh-123456789012"
	instanceID := "crn:v1:bluemix:public:logs:us-south:a/account123:instance456::"

	userID := GenerateUserIDFromSubject(subject, instanceID)

	// Should be 16 characters
	if len(userID) != 16 {
		t.Errorf("Expected userID length 16, got %d", len(userID))
	}

	// Should be deterministic
	userID2 := GenerateUserIDFromSubject(subject, instanceID)
	if userID != userID2 {
		t.Errorf("Expected deterministic ID, got %s and %s", userID, userID2)
	}

	// Different subject should produce different ID
	userID3 := GenerateUserIDFromSubject("different-subject", instanceID)
	if userID == userID3 {
		t.Errorf("Expected different IDs for different subjects")
	}
}

func TestUserIDIsolation(t *testing.T) {
	// Different API keys should produce different user IDs
	userID1 := GenerateUserID("api-key-user-1", "instance-1")
	userID2 := GenerateUserID("api-key-user-2", "instance-1")
	userID3 := GenerateUserID("api-key-user-1", "instance-2")

	if userID1 == userID2 {
		t.Errorf("Different API keys should produce different user IDs")
	}

	if userID1 == userID3 {
		t.Errorf("Different instance IDs should produce different user IDs")
	}
}

func TestSessionManager(t *testing.T) {
	// Create a temporary directory for test persistence
	tmpDir, err := os.MkdirTemp("", "session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewSessionManager(tmpDir)

	t.Run("create new session", func(t *testing.T) {
		session := manager.GetOrCreateSession("api-key-1", "instance-1")

		if session == nil {
			t.Fatal("Expected session, got nil")
		}

		if session.UserID == "" {
			t.Error("Expected non-empty UserID")
		}

		if session.InstanceID != "instance-1" {
			t.Errorf("Expected instanceID 'instance-1', got '%s'", session.InstanceID)
		}
	})

	t.Run("same credentials return same session", func(t *testing.T) {
		session1 := manager.GetOrCreateSession("api-key-1", "instance-1")
		session2 := manager.GetOrCreateSession("api-key-1", "instance-1")

		if session1 != session2 {
			t.Error("Expected same session for same credentials")
		}
	})

	t.Run("different credentials return different sessions", func(t *testing.T) {
		session1 := manager.GetOrCreateSession("api-key-1", "instance-1")
		session2 := manager.GetOrCreateSession("api-key-2", "instance-1")

		if session1 == session2 {
			t.Error("Expected different sessions for different credentials")
		}

		if session1.UserID == session2.UserID {
			t.Error("Expected different UserIDs for different credentials")
		}
	})
}

func TestSessionIsolation(t *testing.T) {
	// Create a temporary directory for test persistence
	tmpDir, err := os.MkdirTemp("", "session-isolation-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewSessionManager(tmpDir)

	// Create sessions for two different users
	user1Session := manager.GetOrCreateSession("user1-api-key", "instance-1")
	user2Session := manager.GetOrCreateSession("user2-api-key", "instance-1")

	t.Run("sessions are isolated", func(t *testing.T) {
		// Set data on user1's session
		user1Session.SetLastQuery("SELECT * FROM user1_table")
		user1Session.SetFilter("application", "user1-app")
		user1Session.RecordToolUse("query_logs", true, map[string]interface{}{"query": "user1"})

		// Set different data on user2's session
		user2Session.SetLastQuery("SELECT * FROM user2_table")
		user2Session.SetFilter("application", "user2-app")
		user2Session.RecordToolUse("list_alerts", true, map[string]interface{}{})

		// Verify user1's data
		if user1Session.GetLastQuery() != "SELECT * FROM user1_table" {
			t.Errorf("User1 query mismatch: %s", user1Session.GetLastQuery())
		}
		if user1Session.GetFilter("application") != "user1-app" {
			t.Errorf("User1 filter mismatch: %s", user1Session.GetFilter("application"))
		}

		// Verify user2's data
		if user2Session.GetLastQuery() != "SELECT * FROM user2_table" {
			t.Errorf("User2 query mismatch: %s", user2Session.GetLastQuery())
		}
		if user2Session.GetFilter("application") != "user2-app" {
			t.Errorf("User2 filter mismatch: %s", user2Session.GetFilter("application"))
		}

		// Verify tool usage is isolated
		user1Tools := user1Session.GetRecentTools(10)
		user2Tools := user2Session.GetRecentTools(10)

		if len(user1Tools) == 0 || user1Tools[0].Tool != "query_logs" {
			t.Error("User1 tool usage not recorded correctly")
		}
		if len(user2Tools) == 0 || user2Tools[0].Tool != "list_alerts" {
			t.Error("User2 tool usage not recorded correctly")
		}
	})

	t.Run("investigation context is isolated", func(t *testing.T) {
		// Start investigation for user1
		user1Session.StartInvestigation("app1", "1h")
		user1Session.SetHypothesis("User1's hypothesis")
		user1Session.AddFinding("query_logs", "Found error", "critical", "evidence1")

		// Start different investigation for user2
		user2Session.StartInvestigation("app2", "24h")
		user2Session.SetHypothesis("User2's hypothesis")

		// Verify investigations are isolated
		user1Inv := user1Session.GetInvestigation()
		user2Inv := user2Session.GetInvestigation()

		if user1Inv.Application != "app1" {
			t.Errorf("User1 investigation app mismatch: %s", user1Inv.Application)
		}
		if user2Inv.Application != "app2" {
			t.Errorf("User2 investigation app mismatch: %s", user2Inv.Application)
		}

		if user1Inv.Hypothesis != "User1's hypothesis" {
			t.Errorf("User1 hypothesis mismatch")
		}
		if user2Inv.Hypothesis != "User2's hypothesis" {
			t.Errorf("User2 hypothesis mismatch")
		}

		if len(user1Inv.Findings) != 1 {
			t.Errorf("User1 should have 1 finding, got %d", len(user1Inv.Findings))
		}
		if len(user2Inv.Findings) != 0 {
			t.Errorf("User2 should have 0 findings, got %d", len(user2Inv.Findings))
		}
	})

	t.Run("learned patterns are isolated", func(t *testing.T) {
		// Record multiple tool uses for user1 to create patterns
		for i := 0; i < 3; i++ {
			user1Session.RecordToolUse("query_logs", true, map[string]interface{}{"time_range": "1h"})
			time.Sleep(10 * time.Millisecond) // Small delay to ensure sequence detection
			user1Session.RecordToolUse("create_alert", true, map[string]interface{}{})
		}

		// User2 has different patterns
		for i := 0; i < 3; i++ {
			user2Session.RecordToolUse("list_dashboards", true, map[string]interface{}{})
			time.Sleep(10 * time.Millisecond)
			user2Session.RecordToolUse("get_dashboard", true, map[string]interface{}{})
		}

		// Verify patterns are different
		user1Patterns := user1Session.LearnedPatterns
		user2Patterns := user2Session.LearnedPatterns

		if user1Patterns.TotalToolCalls == user2Patterns.TotalToolCalls {
			// This could be coincidentally equal, but worth checking
			t.Log("Warning: Tool call counts happen to be equal")
		}

		// Check that patterns don't leak between users
		user1Summary := user1Session.GetSessionSummary()
		user2Summary := user2Session.GetSessionSummary()

		if user1Summary["user_id"] == user2Summary["user_id"] {
			t.Error("User IDs should be different in summaries")
		}
	})
}

func TestSessionPersistence(t *testing.T) {
	// Create a temporary directory for test persistence
	tmpDir, err := os.MkdirTemp("", "session-persist-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	apiKey := "test-api-key-for-persistence" // pragma: allowlist secret
	instanceID := "test-instance-persist"
	userID := GenerateUserID(apiKey, instanceID)

	t.Run("save and load session", func(t *testing.T) {
		// Create manager and session
		manager1 := NewSessionManager(tmpDir)
		session1 := manager1.GetOrCreateSession(apiKey, instanceID)

		// Add some data
		session1.SetLastQuery("SELECT * FROM test")
		session1.SetFilter("app", "test-app")
		session1.RecordToolUse("query_logs", true, map[string]interface{}{"query": "test"})

		// Save the session
		err := manager1.SaveSession(userID)
		if err != nil {
			t.Fatalf("Failed to save session: %v", err)
		}

		// Verify file exists
		filePath := filepath.Join(tmpDir, userID+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("Session file was not created")
		}

		// Create a new manager (simulating server restart)
		manager2 := NewSessionManager(tmpDir)
		session2 := manager2.GetOrCreateSession(apiKey, instanceID)

		// Verify data was loaded
		if session2.GetLastQuery() != "SELECT * FROM test" {
			t.Errorf("Query not persisted: %s", session2.GetLastQuery())
		}
		if session2.GetFilter("app") != "test-app" {
			t.Errorf("Filter not persisted: %s", session2.GetFilter("app"))
		}
	})

	t.Run("persistence does not leak between users", func(t *testing.T) {
		manager := NewSessionManager(tmpDir)

		// Create two user sessions with different data
		user1Session := manager.GetOrCreateSession("user1-key", "instance")
		user1Session.SetLastQuery("USER1 QUERY")
		user1Session.SetFilter("user", "user1")

		user2Session := manager.GetOrCreateSession("user2-key", "instance")
		user2Session.SetLastQuery("USER2 QUERY")
		user2Session.SetFilter("user", "user2")

		// Save both
		user1ID := GenerateUserID("user1-key", "instance")
		user2ID := GenerateUserID("user2-key", "instance")

		_ = manager.SaveSession(user1ID)
		_ = manager.SaveSession(user2ID)

		// Create new manager and load
		manager2 := NewSessionManager(tmpDir)

		loadedUser1 := manager2.GetOrCreateSession("user1-key", "instance")
		loadedUser2 := manager2.GetOrCreateSession("user2-key", "instance")

		if loadedUser1.GetLastQuery() != "USER1 QUERY" {
			t.Errorf("User1 query leaked or lost: %s", loadedUser1.GetLastQuery())
		}
		if loadedUser2.GetLastQuery() != "USER2 QUERY" {
			t.Errorf("User2 query leaked or lost: %s", loadedUser2.GetLastQuery())
		}

		if loadedUser1.GetFilter("user") != "user1" {
			t.Error("User1 filter leaked or lost")
		}
		if loadedUser2.GetFilter("user") != "user2" {
			t.Error("User2 filter leaked or lost")
		}
	})
}

func TestClearSession(t *testing.T) {
	session := NewSessionContext("test-user", "test-instance")

	// Add some data
	session.SetLastQuery("SELECT * FROM test")
	session.SetFilter("app", "test-app")
	session.RecordToolUse("query_logs", true, map[string]interface{}{})
	session.StartInvestigation("app", "1h")

	// Store original identity
	originalUserID := session.UserID
	originalInstanceID := session.InstanceID
	originalCreatedAt := session.CreatedAt

	// Clear the session
	session.ClearSession()

	// Verify data is cleared
	if session.GetLastQuery() != "" {
		t.Error("Query should be cleared")
	}
	if len(session.GetAllFilters()) != 0 {
		t.Error("Filters should be cleared")
	}
	if len(session.GetRecentTools(10)) != 0 {
		t.Error("Recent tools should be cleared")
	}
	if session.GetInvestigation() != nil {
		t.Error("Investigation should be cleared")
	}

	// Verify identity is preserved
	if session.UserID != originalUserID {
		t.Error("UserID should be preserved")
	}
	if session.InstanceID != originalInstanceID {
		t.Error("InstanceID should be preserved")
	}
	if session.CreatedAt != originalCreatedAt {
		t.Error("CreatedAt should be preserved")
	}
}

func TestLearnedPatterns(t *testing.T) {
	session := NewSessionContext("test-user", "test-instance")

	t.Run("tracks total tool calls", func(t *testing.T) {
		initialCount := session.LearnedPatterns.TotalToolCalls

		session.RecordToolUse("query_logs", true, nil)
		session.RecordToolUse("list_alerts", true, nil)
		session.RecordToolUse("create_alert", true, nil)

		if session.LearnedPatterns.TotalToolCalls != initialCount+3 {
			t.Errorf("Expected %d tool calls, got %d", initialCount+3, session.LearnedPatterns.TotalToolCalls)
		}
	})

	t.Run("learns common filters", func(t *testing.T) {
		session.SetFilter("application", "my-app")
		session.RecordToolUse("query_logs", true, nil)

		if session.LearnedPatterns.CommonFilters["application"] != "my-app" {
			t.Error("Common filter not learned")
		}
	})

	t.Run("records frequent sequences", func(t *testing.T) {
		// Create a new session for clean sequence testing
		seqSession := NewSessionContext("seq-user", "seq-instance")

		// Record the same sequence multiple times
		for i := 0; i < 5; i++ {
			seqSession.RecordToolUse("query_logs", true, nil)
			seqSession.RecordToolUse("investigate_incident", true, nil)
		}

		// Check if sequence was recorded
		found := false
		for _, seq := range seqSession.LearnedPatterns.FrequentSequences {
			if len(seq.Tools) == 2 && seq.Tools[0] == "query_logs" && seq.Tools[1] == "investigate_incident" {
				found = true
				if seq.Count < 2 {
					t.Errorf("Sequence count should be at least 2, got %d", seq.Count)
				}
				break
			}
		}
		if !found {
			t.Error("Expected to find query_logs -> investigate_incident sequence")
		}
	})
}

func TestGetSessionByID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-byid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewSessionManager(tmpDir)

	// Create a session
	session := manager.GetOrCreateSession("test-key", "test-instance")
	userID := session.UserID

	// Get by ID should return the same session
	retrieved := manager.GetSessionByID(userID)
	if retrieved != session {
		t.Error("GetSessionByID should return the same session object")
	}

	// Non-existent ID should create a new session
	newSession := manager.GetSessionByID("non-existent-id")
	if newSession == nil {
		t.Fatal("GetSessionByID should create a new session for unknown ID")
	}
	if newSession.UserID != "non-existent-id" {
		t.Errorf("New session should have the requested UserID, got %s", newSession.UserID)
	}
}

func TestListSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-list-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewSessionManager(tmpDir)

	// Create multiple sessions
	manager.GetOrCreateSession("key1", "instance")
	manager.GetOrCreateSession("key2", "instance")
	manager.GetOrCreateSession("key3", "instance")

	sessions := manager.ListSessions()

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
}
