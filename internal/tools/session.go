// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements session context for conversational memory.
package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// validUserIDPattern matches valid user IDs (16 hex characters from SHA256 hash)
var validUserIDPattern = regexp.MustCompile(`^[a-f0-9]{16}$`)

// SessionContext maintains conversational state across tool calls.
// This enables LLMs to reference previous results and maintain context.
type SessionContext struct {
	mu sync.RWMutex

	// UserID uniquely identifies this user (hash of API key + instance ID)
	UserID string `json:"user_id,omitempty"`

	// InstanceID is the IBM Cloud Logs instance ID
	InstanceID string `json:"instance_id,omitempty"`

	// CreatedAt when this session was first created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt when this session was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// LastQuery stores the most recent query executed
	LastQuery string `json:"last_query,omitempty"`

	// LastQueryTime when the last query was executed
	LastQueryTime time.Time `json:"last_query_time,omitempty"`

	// LastResults caches recent tool results (limited to prevent memory bloat)
	LastResults map[string]interface{} `json:"last_results,omitempty"`

	// ActiveFilters stores filters that should persist across queries
	ActiveFilters map[string]string `json:"active_filters,omitempty"`

	// InvestigationContext tracks multi-step investigation state
	InvestigationContext *InvestigationContext `json:"investigation,omitempty"`

	// RecentTools tracks recently used tools for suggestion optimization
	RecentTools []RecentToolUse `json:"recent_tools,omitempty"`

	// Preferences learned from user behavior
	Preferences *UserPreferences `json:"preferences,omitempty"`

	// LearnedPatterns stores persistent patterns across sessions
	LearnedPatterns *LearnedPatterns `json:"learned_patterns,omitempty"`
}

// LearnedPatterns stores patterns that persist across sessions
type LearnedPatterns struct {
	// FrequentSequences are tool sequences used frequently
	FrequentSequences []PatternSequence `json:"frequent_sequences,omitempty"`

	// PreferredWorkflows are complete workflows the user commonly executes
	PreferredWorkflows []string `json:"preferred_workflows,omitempty"`

	// CommonFilters are filters the user applies repeatedly
	CommonFilters map[string]string `json:"common_filters,omitempty"`

	// TotalToolCalls lifetime tool call count
	TotalToolCalls int `json:"total_tool_calls"`

	// LastUpdated when patterns were last updated
	LastUpdated time.Time `json:"last_updated"`
}

// PatternSequence represents a learned tool sequence pattern
type PatternSequence struct {
	Tools       []string  `json:"tools"`
	Count       int       `json:"count"`
	SuccessRate float64   `json:"success_rate"`
	LastUsed    time.Time `json:"last_used"`
}

// InvestigationContext tracks state during multi-step investigations
type InvestigationContext struct {
	// ID unique identifier for this investigation
	ID string `json:"id"`

	// StartTime when investigation began
	StartTime time.Time `json:"start_time"`

	// Application being investigated
	Application string `json:"application,omitempty"`

	// TimeRange for the investigation
	TimeRange string `json:"time_range,omitempty"`

	// Hypothesis current working hypothesis
	Hypothesis string `json:"hypothesis,omitempty"`

	// Findings accumulated during investigation
	Findings []Finding `json:"findings,omitempty"`

	// ToolsUsed tracks tools used in this investigation
	ToolsUsed []string `json:"tools_used,omitempty"`
}

// Finding represents a discovery during investigation
type Finding struct {
	Timestamp   time.Time `json:"timestamp"`
	Tool        string    `json:"tool"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // info, warning, critical
	Evidence    string    `json:"evidence,omitempty"`
}

// RecentToolUse tracks a recent tool invocation
type RecentToolUse struct {
	Tool      string                 `json:"tool"`
	Timestamp time.Time              `json:"timestamp"`
	Success   bool                   `json:"success"`
	Args      map[string]interface{} `json:"args,omitempty"`
}

// UserPreferences tracks learned user preferences
type UserPreferences struct {
	// PreferredTimeRange default time range for queries
	PreferredTimeRange string `json:"preferred_time_range,omitempty"`

	// PreferredSeverity minimum severity level of interest
	PreferredSeverity int `json:"preferred_severity,omitempty"`

	// FrequentApplications commonly queried applications
	FrequentApplications []string `json:"frequent_applications,omitempty"`

	// PreferredLimit default result limit
	PreferredLimit int `json:"preferred_limit,omitempty"`
}

// SessionManager manages user-specific sessions with persistence
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionContext // keyed by userID
	dataDir  string                     // directory for persistence
}

// Global session manager (singleton for MCP server lifecycle)
var (
	globalSessionManager     *SessionManager
	globalSessionManagerOnce sync.Once
	currentUserID            string // set during initialization
)

// GetSessionManager returns the global session manager
func GetSessionManager() *SessionManager {
	globalSessionManagerOnce.Do(func() {
		globalSessionManager = NewSessionManager("")
	})
	return globalSessionManager
}

// NewSessionManager creates a new session manager
func NewSessionManager(dataDir string) *SessionManager {
	if dataDir == "" {
		// Default to user's config directory
		homeDir, err := os.UserHomeDir()
		if err == nil {
			dataDir = filepath.Join(homeDir, ".logs-mcp", "sessions")
		}
	}
	return &SessionManager{
		sessions: make(map[string]*SessionContext),
		dataDir:  dataDir,
	}
}

// SetCurrentUser sets the current user context (called during initialization)
// Uses hashed API key + instance ID as fallback when JWT is not available
func SetCurrentUser(apiKey, instanceID string) {
	currentUserID = GenerateUserID(apiKey, instanceID)
	// Pre-load or create session for this user
	GetSessionManager().GetOrCreateSession(apiKey, instanceID)
}

// SetCurrentUserFromJWT sets the current user from JWT subject claim
// The subject is the unique identifier from the IAM token (e.g., "iam-ServiceId-...")
func SetCurrentUserFromJWT(jwtSubject, instanceID string) {
	// Use the JWT subject directly as the user ID (it's already unique)
	// Hash it to create a safe filename for persistence
	currentUserID = GenerateUserIDFromSubject(jwtSubject, instanceID)
	// Pre-load or create session for this user
	GetSessionManager().GetOrCreateSessionByID(currentUserID, instanceID)
}

// GenerateUserIDFromSubject creates a user ID from JWT subject and instance ID
func GenerateUserIDFromSubject(subject, instanceID string) string {
	h := sha256.New()
	h.Write([]byte(subject + ":" + instanceID))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// GenerateUserID creates a unique user identifier from API key and instance ID
func GenerateUserID(apiKey, instanceID string) string {
	// Use first 8 chars of SHA256 hash for privacy (don't store full key)
	h := sha256.New()
	h.Write([]byte(apiKey + ":" + instanceID))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// GetSession returns the session for the current user (backward compatible)
func GetSession() *SessionContext {
	if currentUserID != "" {
		return GetSessionManager().GetSessionByID(currentUserID)
	}
	// Fallback: return a default session if no user set
	return GetSessionManager().GetOrCreateSession("", "default")
}

// GetOrCreateSession returns an existing session or creates a new one
func (m *SessionManager) GetOrCreateSession(apiKey, instanceID string) *SessionContext {
	userID := GenerateUserID(apiKey, instanceID)
	return m.GetOrCreateSessionByID(userID, instanceID)
}

// GetOrCreateSessionByID returns an existing session or creates a new one using a pre-computed user ID
func (m *SessionManager) GetOrCreateSessionByID(userID, instanceID string) *SessionContext {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[userID]; exists {
		return session
	}

	// Try to load from disk first
	session := m.loadSession(userID)
	if session == nil {
		session = NewSessionContext(userID, instanceID)
	}

	m.sessions[userID] = session
	return session
}

// GetSessionByID returns a session by user ID
func (m *SessionManager) GetSessionByID(userID string) *SessionContext {
	m.mu.RLock()
	session, exists := m.sessions[userID]
	m.mu.RUnlock()

	if exists {
		return session
	}

	// Session not loaded, try to load from disk
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if session, exists := m.sessions[userID]; exists {
		return session
	}

	session = m.loadSession(userID)
	if session == nil {
		// Create a minimal session
		session = &SessionContext{
			UserID:        userID,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			LastResults:   make(map[string]interface{}),
			ActiveFilters: make(map[string]string),
			RecentTools:   make([]RecentToolUse, 0, 20),
			Preferences:   &UserPreferences{},
			LearnedPatterns: &LearnedPatterns{
				CommonFilters: make(map[string]string),
			},
		}
	}

	m.sessions[userID] = session
	return session
}

// isValidUserID validates that a user ID is safe for use in file paths.
// User IDs must be exactly 16 lowercase hex characters (from SHA256 hash).
// This prevents path traversal attacks and other injection attempts.
func isValidUserID(userID string) bool {
	return validUserIDPattern.MatchString(userID)
}

// loadSession loads a session from disk
func (m *SessionManager) loadSession(userID string) *SessionContext {
	if m.dataDir == "" {
		return nil
	}

	// Validate userID to prevent path traversal attacks
	if !isValidUserID(userID) {
		return nil
	}

	filePath := filepath.Join(m.dataDir, userID+".json")
	data, err := os.ReadFile(filePath) // #nosec G304 -- userID is validated above
	if err != nil {
		return nil // File doesn't exist or can't be read
	}

	var session SessionContext
	if err := json.Unmarshal(data, &session); err != nil {
		return nil
	}

	// Reinitialize maps that might be nil after unmarshal
	if session.LastResults == nil {
		session.LastResults = make(map[string]interface{})
	}
	if session.ActiveFilters == nil {
		session.ActiveFilters = make(map[string]string)
	}
	if session.RecentTools == nil {
		session.RecentTools = make([]RecentToolUse, 0, 20)
	}
	if session.Preferences == nil {
		session.Preferences = &UserPreferences{}
	}
	if session.LearnedPatterns == nil {
		session.LearnedPatterns = &LearnedPatterns{
			CommonFilters: make(map[string]string),
		}
	}

	return &session
}

// SaveSession persists a session to disk
func (m *SessionManager) SaveSession(userID string) error {
	// Validate userID to prevent path traversal attacks
	if !isValidUserID(userID) {
		return nil // Invalid userID, don't save
	}

	m.mu.RLock()
	session, exists := m.sessions[userID]
	m.mu.RUnlock()

	if !exists {
		return nil // Nothing to save
	}

	if m.dataDir == "" {
		return nil // Persistence disabled
	}

	// Ensure directory exists
	if err := os.MkdirAll(m.dataDir, 0700); err != nil {
		return err
	}

	session.mu.RLock()
	session.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(session, "", "  ")
	session.mu.RUnlock()

	if err != nil {
		return err
	}

	filePath := filepath.Join(m.dataDir, userID+".json")
	return os.WriteFile(filePath, data, 0600)
}

// SaveCurrentSession saves the current user's session
func SaveCurrentSession() error {
	if currentUserID == "" {
		return nil
	}
	return GetSessionManager().SaveSession(currentUserID)
}

// ListSessions returns all active session IDs
func (m *SessionManager) ListSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

// NewSessionContext creates a new session context for a user
func NewSessionContext(userID, instanceID string) *SessionContext {
	now := time.Now()
	return &SessionContext{
		UserID:        userID,
		InstanceID:    instanceID,
		CreatedAt:     now,
		UpdatedAt:     now,
		LastResults:   make(map[string]interface{}),
		ActiveFilters: make(map[string]string),
		RecentTools:   make([]RecentToolUse, 0, 20),
		Preferences:   &UserPreferences{},
		LearnedPatterns: &LearnedPatterns{
			CommonFilters: make(map[string]string),
		},
	}
}

// SetLastQuery records the last executed query
func (s *SessionContext) SetLastQuery(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastQuery = query
	s.LastQueryTime = time.Now()
}

// GetLastQuery returns the last executed query
func (s *SessionContext) GetLastQuery() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastQuery
}

// SetFilter sets a persistent filter
func (s *SessionContext) SetFilter(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveFilters[key] = value
}

// GetFilter retrieves a persistent filter
func (s *SessionContext) GetFilter(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ActiveFilters[key]
}

// GetAllFilters returns all active filters
func (s *SessionContext) GetAllFilters() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	filters := make(map[string]string, len(s.ActiveFilters))
	for k, v := range s.ActiveFilters {
		filters[k] = v
	}
	return filters
}

// ClearFilters removes all active filters
func (s *SessionContext) ClearFilters() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveFilters = make(map[string]string)
}

// ClearSession resets all session state while preserving user identity
func (s *SessionContext) ClearSession() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Preserve identity
	userID := s.UserID
	instanceID := s.InstanceID
	createdAt := s.CreatedAt

	// Reset all state
	s.LastQuery = ""
	s.LastQueryTime = time.Time{}
	s.LastResults = make(map[string]interface{})
	s.ActiveFilters = make(map[string]string)
	s.InvestigationContext = nil
	s.RecentTools = make([]RecentToolUse, 0, 20)
	s.Preferences = &UserPreferences{}
	s.LearnedPatterns = &LearnedPatterns{
		CommonFilters: make(map[string]string),
	}

	// Restore identity
	s.UserID = userID
	s.InstanceID = instanceID
	s.CreatedAt = createdAt
	s.UpdatedAt = time.Now()
}

// CacheResult stores a tool result for later reference
func (s *SessionContext) CacheResult(toolName string, result map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store with tool name as key, limit cache size
	s.LastResults[toolName] = result

	// Limit cache to last 5 tool results
	if len(s.LastResults) > 5 {
		// Remove oldest (simple approach - just clear one arbitrary entry)
		for k := range s.LastResults {
			if k != toolName {
				delete(s.LastResults, k)
				break
			}
		}
	}
}

// GetCachedResult retrieves a cached tool result
func (s *SessionContext) GetCachedResult(toolName string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if result, ok := s.LastResults[toolName]; ok {
		if m, ok := result.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// RecordToolUse records a tool invocation
func (s *SessionContext) RecordToolUse(toolName string, success bool, args map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	use := RecentToolUse{
		Tool:      toolName,
		Timestamp: time.Now(),
		Success:   success,
		Args:      args,
	}

	s.RecentTools = append(s.RecentTools, use)

	// Keep only last 20 tool uses
	if len(s.RecentTools) > 20 {
		s.RecentTools = s.RecentTools[len(s.RecentTools)-20:]
	}

	// Learn preferences from usage
	s.learnPreferences(toolName, args)

	// Update learned patterns for persistence
	s.updateLearnedPatterns(toolName, success)

	// Mark session as updated
	s.UpdatedAt = time.Now()
}

// updateLearnedPatterns updates persistent learned patterns
func (s *SessionContext) updateLearnedPatterns(_ string, success bool) {
	if s.LearnedPatterns == nil {
		s.LearnedPatterns = &LearnedPatterns{
			CommonFilters: make(map[string]string),
		}
	}

	s.LearnedPatterns.TotalToolCalls++
	s.LearnedPatterns.LastUpdated = time.Now()

	// Learn common filters
	for key, value := range s.ActiveFilters {
		if existing, ok := s.LearnedPatterns.CommonFilters[key]; ok {
			// Keep the most recent value
			if value != existing {
				s.LearnedPatterns.CommonFilters[key] = value
			}
		} else {
			s.LearnedPatterns.CommonFilters[key] = value
		}
	}

	// Learn tool sequences (requires at least 2 tools)
	if len(s.RecentTools) >= 2 {
		// Look at the last pair
		prev := s.RecentTools[len(s.RecentTools)-2]
		curr := s.RecentTools[len(s.RecentTools)-1]

		// Only count sequences within 5 minutes
		if curr.Timestamp.Sub(prev.Timestamp) <= 5*time.Minute {
			sequence := []string{prev.Tool, curr.Tool}
			s.recordSequence(sequence, prev.Success && success)
		}
	}
}

// recordSequence records or updates a tool sequence pattern
func (s *SessionContext) recordSequence(tools []string, success bool) {
	// Find existing sequence
	for i, seq := range s.LearnedPatterns.FrequentSequences {
		if len(seq.Tools) == len(tools) {
			match := true
			for j, t := range tools {
				if seq.Tools[j] != t {
					match = false
					break
				}
			}
			if match {
				// Update existing sequence
				s.LearnedPatterns.FrequentSequences[i].Count++
				s.LearnedPatterns.FrequentSequences[i].LastUsed = time.Now()
				// Update success rate (rolling average)
				oldRate := s.LearnedPatterns.FrequentSequences[i].SuccessRate
				count := float64(s.LearnedPatterns.FrequentSequences[i].Count)
				successVal := 0.0
				if success {
					successVal = 100.0
				}
				s.LearnedPatterns.FrequentSequences[i].SuccessRate = ((oldRate * (count - 1)) + successVal) / count
				return
			}
		}
	}

	// Add new sequence
	successRate := 0.0
	if success {
		successRate = 100.0
	}
	s.LearnedPatterns.FrequentSequences = append(s.LearnedPatterns.FrequentSequences, PatternSequence{
		Tools:       tools,
		Count:       1,
		SuccessRate: successRate,
		LastUsed:    time.Now(),
	})

	// Keep only top 20 sequences by count
	if len(s.LearnedPatterns.FrequentSequences) > 20 {
		// Sort by count descending
		seqs := s.LearnedPatterns.FrequentSequences
		for i := 0; i < len(seqs); i++ {
			for j := i + 1; j < len(seqs); j++ {
				if seqs[j].Count > seqs[i].Count {
					seqs[i], seqs[j] = seqs[j], seqs[i]
				}
			}
		}
		s.LearnedPatterns.FrequentSequences = seqs[:20]
	}
}

// learnPreferences updates user preferences based on tool usage
func (s *SessionContext) learnPreferences(_ string, args map[string]interface{}) {
	if s.Preferences == nil {
		s.Preferences = &UserPreferences{}
	}

	// Learn time range preference
	if timeRange, ok := args["time_range"].(string); ok && timeRange != "" {
		s.Preferences.PreferredTimeRange = timeRange
	}

	// Learn application preference
	if app, ok := args["application"].(string); ok && app != "" {
		s.addFrequentApplication(app)
	}
	if app, ok := args["applicationName"].(string); ok && app != "" {
		s.addFrequentApplication(app)
	}

	// Learn limit preference
	if limit, ok := args["limit"].(float64); ok && limit > 0 {
		s.Preferences.PreferredLimit = int(limit)
	}
}

// addFrequentApplication adds an application to the frequent list
func (s *SessionContext) addFrequentApplication(app string) {
	for _, existing := range s.Preferences.FrequentApplications {
		if existing == app {
			return // Already in list
		}
	}
	s.Preferences.FrequentApplications = append(s.Preferences.FrequentApplications, app)
	// Keep only top 5
	if len(s.Preferences.FrequentApplications) > 5 {
		s.Preferences.FrequentApplications = s.Preferences.FrequentApplications[1:]
	}
}

// GetRecentTools returns recently used tools
func (s *SessionContext) GetRecentTools(limit int) []RecentToolUse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.RecentTools) {
		limit = len(s.RecentTools)
	}

	result := make([]RecentToolUse, limit)
	copy(result, s.RecentTools[len(s.RecentTools)-limit:])
	return result
}

// StartInvestigation begins a new investigation context
func (s *SessionContext) StartInvestigation(application, timeRange string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := time.Now().Format("20060102-150405")
	s.InvestigationContext = &InvestigationContext{
		ID:          id,
		StartTime:   time.Now(),
		Application: application,
		TimeRange:   timeRange,
		Findings:    []Finding{},
		ToolsUsed:   []string{},
	}

	return id
}

// AddFinding adds a finding to the current investigation
func (s *SessionContext) AddFinding(tool, description, severity, evidence string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.InvestigationContext == nil {
		return
	}

	s.InvestigationContext.Findings = append(s.InvestigationContext.Findings, Finding{
		Timestamp:   time.Now(),
		Tool:        tool,
		Description: description,
		Severity:    severity,
		Evidence:    evidence,
	})

	// Track tool usage
	s.InvestigationContext.ToolsUsed = append(s.InvestigationContext.ToolsUsed, tool)
}

// SetHypothesis sets the current working hypothesis
func (s *SessionContext) SetHypothesis(hypothesis string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.InvestigationContext != nil {
		s.InvestigationContext.Hypothesis = hypothesis
	}
}

// GetInvestigation returns the current investigation context
func (s *SessionContext) GetInvestigation() *InvestigationContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.InvestigationContext
}

// EndInvestigation clears the investigation context
func (s *SessionContext) EndInvestigation() *InvestigationContext {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv := s.InvestigationContext
	s.InvestigationContext = nil
	return inv
}

// GetPreferences returns user preferences
func (s *SessionContext) GetPreferences() *UserPreferences {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Preferences
}

// ApplySessionDefaults applies session preferences to arguments if not already set
func (s *SessionContext) ApplySessionDefaults(args map[string]interface{}) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if args == nil {
		args = make(map[string]interface{})
	}

	if s.Preferences == nil {
		return args
	}

	// Apply time range default
	if _, ok := args["time_range"]; !ok && s.Preferences.PreferredTimeRange != "" {
		args["time_range"] = s.Preferences.PreferredTimeRange
	}

	// Apply limit default
	if _, ok := args["limit"]; !ok && s.Preferences.PreferredLimit > 0 {
		args["limit"] = s.Preferences.PreferredLimit
	}

	// Apply application from active filters
	if _, ok := args["application"]; !ok {
		if app, exists := s.ActiveFilters["application"]; exists {
			args["application"] = app
		}
	}

	return args
}

// GetSessionSummary returns a summary of the current session state
func (s *SessionContext) GetSessionSummary() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := map[string]interface{}{
		"has_active_filters": len(s.ActiveFilters) > 0,
		"active_filters":     s.ActiveFilters,
		"recent_tools_count": len(s.RecentTools),
	}

	// Add user identity info
	if s.UserID != "" {
		summary["user_id"] = s.UserID
		summary["instance_id"] = s.InstanceID
		summary["session_created"] = s.CreatedAt.Format(time.RFC3339)
		summary["session_age"] = time.Since(s.CreatedAt).String()
	}

	if s.LastQuery != "" {
		summary["last_query"] = s.LastQuery
		summary["last_query_age"] = time.Since(s.LastQueryTime).String()
	}

	if s.InvestigationContext != nil {
		summary["active_investigation"] = map[string]interface{}{
			"id":             s.InvestigationContext.ID,
			"application":    s.InvestigationContext.Application,
			"findings_count": len(s.InvestigationContext.Findings),
			"hypothesis":     s.InvestigationContext.Hypothesis,
		}
	}

	if s.Preferences != nil {
		summary["learned_preferences"] = s.Preferences
	}

	// Add persistent learned patterns
	if s.LearnedPatterns != nil {
		summary["learned_patterns"] = map[string]interface{}{
			"total_tool_calls":     s.LearnedPatterns.TotalToolCalls,
			"frequent_sequences":   len(s.LearnedPatterns.FrequentSequences),
			"common_filters":       s.LearnedPatterns.CommonFilters,
			"preferred_workflows":  s.LearnedPatterns.PreferredWorkflows,
			"last_patterns_update": s.LearnedPatterns.LastUpdated.Format(time.RFC3339),
		}
	}

	// Add tool usage analytics
	summary["tool_analytics"] = s.getToolAnalyticsLocked()

	return summary
}

// GetToolAnalytics returns analytics about tool usage patterns
func (s *SessionContext) GetToolAnalytics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getToolAnalyticsLocked()
}

// getToolAnalyticsLocked returns tool analytics (caller must hold lock)
func (s *SessionContext) getToolAnalyticsLocked() map[string]interface{} {
	if len(s.RecentTools) == 0 {
		return map[string]interface{}{
			"total_uses":   0,
			"success_rate": 0.0,
			"most_used":    "",
			"tool_counts":  map[string]int{},
			"error_tools":  []string{},
		}
	}

	// Count tool uses and successes
	toolCounts := make(map[string]int)
	toolSuccesses := make(map[string]int)
	errorTools := make(map[string]bool)
	totalSuccess := 0

	for _, use := range s.RecentTools {
		toolCounts[use.Tool]++
		if use.Success {
			toolSuccesses[use.Tool]++
			totalSuccess++
		} else {
			errorTools[use.Tool] = true
		}
	}

	// Find most used tool
	mostUsed := ""
	maxCount := 0
	for tool, count := range toolCounts {
		if count > maxCount {
			maxCount = count
			mostUsed = tool
		}
	}

	// Build error tools list
	errorToolsList := make([]string, 0, len(errorTools))
	for tool := range errorTools {
		errorToolsList = append(errorToolsList, tool)
	}

	// Calculate per-tool success rates
	toolSuccessRates := make(map[string]float64)
	for tool, count := range toolCounts {
		if count > 0 {
			toolSuccessRates[tool] = float64(toolSuccesses[tool]) * 100 / float64(count)
		}
	}

	return map[string]interface{}{
		"total_uses":         len(s.RecentTools),
		"success_rate":       float64(totalSuccess) * 100 / float64(len(s.RecentTools)),
		"most_used":          mostUsed,
		"most_used_count":    maxCount,
		"tool_counts":        toolCounts,
		"tool_success_rates": toolSuccessRates,
		"error_tools":        errorToolsList,
		"learned_sequences":  s.getLearnedSequencesLocked(),
	}
}

// ToolSequence represents a learned sequence of tool invocations
type ToolSequence struct {
	Sequence []string `json:"sequence"`
	Count    int      `json:"count"`
	Success  bool     `json:"success"`
}

// getLearnedSequencesLocked analyzes tool usage to find common patterns
func (s *SessionContext) getLearnedSequencesLocked() []map[string]interface{} {
	if len(s.RecentTools) < 2 {
		return nil
	}

	// Track consecutive tool pairs and their success
	pairCounts := make(map[string]int)
	pairSuccess := make(map[string]int)

	for i := 0; i < len(s.RecentTools)-1; i++ {
		current := s.RecentTools[i]
		next := s.RecentTools[i+1]

		// Check if within 5 minutes of each other (same workflow)
		if next.Timestamp.Sub(current.Timestamp) <= 5*time.Minute {
			pair := current.Tool + " -> " + next.Tool
			pairCounts[pair]++
			if current.Success && next.Success {
				pairSuccess[pair]++
			}
		}
	}

	// Convert to slice and sort by count
	var sequences []map[string]interface{}
	for pair, count := range pairCounts {
		if count >= 2 { // Only include patterns seen at least twice
			sequences = append(sequences, map[string]interface{}{
				"pattern":      pair,
				"count":        count,
				"success_rate": float64(pairSuccess[pair]) * 100 / float64(count),
			})
		}
	}

	// Sort by count descending
	for i := 0; i < len(sequences); i++ {
		for j := i + 1; j < len(sequences); j++ {
			if sequences[j]["count"].(int) > sequences[i]["count"].(int) {
				sequences[i], sequences[j] = sequences[j], sequences[i]
			}
		}
	}

	// Return top 5
	if len(sequences) > 5 {
		sequences = sequences[:5]
	}

	return sequences
}

// GetSuggestedNextTools returns tools likely to be used next based on learned patterns
func (s *SessionContext) GetSuggestedNextTools() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.RecentTools) == 0 {
		return nil
	}

	lastTool := s.RecentTools[len(s.RecentTools)-1].Tool
	suggestions := make(map[string]int)

	// Look for patterns where lastTool was followed by another tool
	for i := 0; i < len(s.RecentTools)-1; i++ {
		if s.RecentTools[i].Tool == lastTool && s.RecentTools[i].Success {
			next := s.RecentTools[i+1]
			// Check if within reasonable time window
			if next.Timestamp.Sub(s.RecentTools[i].Timestamp) <= 5*time.Minute {
				suggestions[next.Tool]++
			}
		}
	}

	// Convert to slice sorted by frequency
	var result []string
	type toolCount struct {
		tool  string
		count int
	}
	var counts []toolCount
	for tool, count := range suggestions {
		counts = append(counts, toolCount{tool, count})
	}

	// Sort by count
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	for _, tc := range counts {
		if len(result) >= 3 {
			break
		}
		result = append(result, tc.tool)
	}

	return result
}

// GetWorkflowSuggestion suggests a workflow based on the current context
func (s *SessionContext) GetWorkflowSuggestion() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	suggestion := map[string]interface{}{
		"has_suggestion": false,
	}

	// Check if there's an active investigation
	if s.InvestigationContext != nil {
		suggestion["has_suggestion"] = true
		suggestion["context"] = "active_investigation"
		suggestion["message"] = "You have an active investigation. Consider continuing with these tools:"

		// Suggest based on what hasn't been used yet in the investigation
		usedTools := make(map[string]bool)
		for _, t := range s.InvestigationContext.ToolsUsed {
			usedTools[t] = true
		}

		var nextSteps []string
		investigationTools := []string{"query_logs", "list_alerts", "suggest_alert", "create_dashboard"}
		for _, tool := range investigationTools {
			if !usedTools[tool] {
				nextSteps = append(nextSteps, tool)
			}
		}
		suggestion["suggested_tools"] = nextSteps
		suggestion["findings_count"] = len(s.InvestigationContext.Findings)
		return suggestion
	}

	// Check recent tool patterns
	if len(s.RecentTools) > 0 {
		lastTool := s.RecentTools[len(s.RecentTools)-1]

		// Suggest based on last tool used
		workflowHints := map[string][]string{
			"query_logs":           {"investigate_incident", "create_alert", "create_dashboard"},
			"investigate_incident": {"suggest_alert", "create_alert", "query_logs"},
			"list_alerts":          {"create_alert", "create_outgoing_webhook"},
			"list_dashboards":      {"create_dashboard", "get_dashboard"},
			"health_check":         {"query_logs", "investigate_incident", "list_alerts"},
		}

		if hints, ok := workflowHints[lastTool.Tool]; ok {
			suggestion["has_suggestion"] = true
			suggestion["context"] = "tool_continuation"
			suggestion["last_tool"] = lastTool.Tool
			suggestion["suggested_tools"] = hints
			suggestion["message"] = "Based on your last action, you might want to:"
		}
	}

	return suggestion
}
