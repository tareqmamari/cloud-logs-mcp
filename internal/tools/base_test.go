package tools

import (
	"testing"
)

func TestGetStringParam(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]interface{}
		key       string
		required  bool
		want      string
		wantErr   bool
	}{
		{
			name:      "valid string parameter",
			arguments: map[string]interface{}{"id": "test-123"},
			key:       "id",
			required:  true,
			want:      "test-123",
			wantErr:   false,
		},
		{
			name:      "missing required parameter",
			arguments: map[string]interface{}{},
			key:       "id",
			required:  true,
			want:      "",
			wantErr:   true,
		},
		{
			name:      "missing optional parameter",
			arguments: map[string]interface{}{},
			key:       "id",
			required:  false,
			want:      "",
			wantErr:   false,
		},
		{
			name:      "numeric ID converted to string",
			arguments: map[string]interface{}{"id": 123},
			key:       "id",
			required:  true,
			want:      "123",
			wantErr:   false,
		},
		{
			name:      "float64 ID converted to string",
			arguments: map[string]interface{}{"id": float64(456)},
			key:       "id",
			required:  true,
			want:      "456",
			wantErr:   false,
		},
		{
			name:      "truly wrong type (map)",
			arguments: map[string]interface{}{"id": map[string]interface{}{"key": "value"}},
			key:       "id",
			required:  true,
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStringParam(tt.arguments, tt.key, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStringParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetStringParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIntParam(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]interface{}
		key       string
		required  bool
		want      int
		wantErr   bool
	}{
		{
			name:      "valid int from float64",
			arguments: map[string]interface{}{"limit": float64(100)},
			key:       "limit",
			required:  true,
			want:      100,
			wantErr:   false,
		},
		{
			name:      "valid int",
			arguments: map[string]interface{}{"limit": 100},
			key:       "limit",
			required:  true,
			want:      100,
			wantErr:   false,
		},
		{
			name:      "missing required",
			arguments: map[string]interface{}{},
			key:       "limit",
			required:  true,
			want:      0,
			wantErr:   true,
		},
		{
			name:      "wrong type",
			arguments: map[string]interface{}{"limit": "not-a-number"},
			key:       "limit",
			required:  true,
			want:      0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetIntParam(tt.arguments, tt.key, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIntParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetIntParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetObjectParam(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]interface{}
		key       string
		required  bool
		wantNil   bool
		wantErr   bool
	}{
		{
			name:      "valid object",
			arguments: map[string]interface{}{"config": map[string]interface{}{"key": "value"}},
			key:       "config",
			required:  true,
			wantNil:   false,
			wantErr:   false,
		},
		{
			name:      "missing required object",
			arguments: map[string]interface{}{},
			key:       "config",
			required:  true,
			wantNil:   true,
			wantErr:   true,
		},
		{
			name:      "wrong type",
			arguments: map[string]interface{}{"config": "not-an-object"},
			key:       "config",
			required:  true,
			wantNil:   true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetObjectParam(tt.arguments, tt.key, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetObjectParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("GetObjectParam() nil = %v, want %v", got == nil, tt.wantNil)
			}
		})
	}
}

func TestGetBoolParam(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]interface{}
		key       string
		required  bool
		want      bool
		wantErr   bool
	}{
		{
			name:      "valid true",
			arguments: map[string]interface{}{"enabled": true},
			key:       "enabled",
			required:  true,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "valid false",
			arguments: map[string]interface{}{"enabled": false},
			key:       "enabled",
			required:  true,
			want:      false,
			wantErr:   false,
		},
		{
			name:      "missing optional",
			arguments: map[string]interface{}{},
			key:       "enabled",
			required:  false,
			want:      false,
			wantErr:   false,
		},
		{
			name:      "string true",
			arguments: map[string]interface{}{"enabled": "true"},
			key:       "enabled",
			required:  true,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "truly wrong type",
			arguments: map[string]interface{}{"enabled": 123},
			key:       "enabled",
			required:  true,
			want:      false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBoolParam(tt.arguments, tt.key, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBoolParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBoolParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPaginationParams(t *testing.T) {
	tests := []struct {
		name       string
		arguments  map[string]interface{}
		wantLimit  int
		wantCursor string
		wantErr    bool
	}{
		{
			name:       "default values",
			arguments:  map[string]interface{}{},
			wantLimit:  0, // Now returns empty map if not present
			wantCursor: "",
			wantErr:    false,
		},
		{
			name: "custom limit",
			arguments: map[string]interface{}{
				"limit": float64(25),
			},
			wantLimit:  25,
			wantCursor: "",
			wantErr:    false,
		},
		{
			name: "with cursor",
			arguments: map[string]interface{}{
				"limit":  float64(25),
				"cursor": "next-page-token",
			},
			wantLimit:  25,
			wantCursor: "next-page-token",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPaginationParams(tt.arguments)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPaginationParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if limit, ok := got["limit"]; ok {
					if l, ok := limit.(float64); ok {
						if int(l) != tt.wantLimit {
							t.Errorf("Limit = %v, want %v", l, tt.wantLimit)
						}
					}
				}
				if cursor, ok := got["cursor"]; ok {
					if c, ok := cursor.(string); ok {
						if c != tt.wantCursor {
							t.Errorf("Cursor = %v, want %v", c, tt.wantCursor)
						}
					}
				}
			}
		})
	}
}

func TestAddPaginationToQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      map[string]string
		params     map[string]interface{}
		wantLimit  string
		wantCursor string
	}{
		{
			name:  "add both params",
			query: map[string]string{},
			params: map[string]interface{}{
				"limit":  25,
				"cursor": "token-123",
			},
			wantLimit:  "25",
			wantCursor: "token-123",
		},
		{
			name:  "only limit",
			query: map[string]string{},
			params: map[string]interface{}{
				"limit": 50,
			},
			wantLimit:  "50",
			wantCursor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddPaginationToQuery(tt.query, tt.params)

			if tt.wantLimit != "" && tt.query["limit"] != tt.wantLimit {
				t.Errorf("limit = %v, want %v", tt.query["limit"], tt.wantLimit)
			}
			if tt.wantCursor != "" && tt.query["cursor"] != tt.wantCursor {
				t.Errorf("cursor = %v, want %v", tt.query["cursor"], tt.wantCursor)
			}
		})
	}
}
