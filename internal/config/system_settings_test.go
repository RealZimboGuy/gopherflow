package config

import (
	"testing"
)

func TestGetSystemSettingStringPrefersTheEnvironment(t *testing.T) {
	t.Setenv(ENGINE_BATCH_SIZE, "99")

	if got := GetSystemSettingString(ENGINE_BATCH_SIZE); got != "99" {
		t.Errorf("got %q, want the environment value 99", got)
	}
}

func TestGetSystemSettingStringDefaults(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{ENGINE_CHECK_DB_INTERVAL, "3s"},
		{ENGINE_STUCK_WORKFLOWS_INTERVAL, "60s"},
		{ENGINE_BATCH_SIZE, "15"},
		{ENGINE_STUCK_WORKFLOWS_REPAIR_AFTER_MINUTES, "5"},
		{ENGINE_EXECUTOR_SIZE, "15"},
		{ENGINE_EXECUTOR_GROUP, "default"},
		{ENGINE_SERVER_WEB_PORT, "8080"},
		{WEB_SESSION_EXPIRY_HOURS, "1"},
		{DATABASE_SQLLITE_FILE_NAME, "./gflow.db"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Setenv(tt.key, "")
			if got := GetSystemSettingString(tt.key); got != tt.want {
				t.Errorf("GetSystemSettingString(%s) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetSystemSettingStringUnknownKeyIsEmpty(t *testing.T) {
	if got := GetSystemSettingString("GFLOW_NOT_A_REAL_SETTING"); got != "" {
		t.Errorf("got %q, want an empty string for an unknown key", got)
	}
}

func TestGetSystemSettingInteger(t *testing.T) {
	t.Setenv(ENGINE_BATCH_SIZE, "42")

	if got := GetSystemSettingInteger(ENGINE_BATCH_SIZE); got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestGetSystemSettingIntegerUnknownKeyIsZero(t *testing.T) {
	if got := GetSystemSettingInteger("GFLOW_NOT_A_REAL_SETTING"); got != 0 {
		t.Errorf("got %d, want 0 for an unset key with no default", got)
	}
}

func TestGetSystemSettingIntegerPanicsOnANonInteger(t *testing.T) {
	t.Setenv(ENGINE_BATCH_SIZE, "not-a-number")

	defer func() {
		if recover() == nil {
			t.Error("a non-integer setting was accepted; want a panic so misconfiguration fails at startup")
		}
	}()

	_ = GetSystemSettingInteger(ENGINE_BATCH_SIZE)
}
