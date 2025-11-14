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
			name:      "wrong type",
			arguments: map[string]interface{}{"id": 123},
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
			name:      "wrong type",
			arguments: map[string]interface{}{"enabled": "true"},
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
