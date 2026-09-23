package main

import (
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "negative max retries",
			config: Config{
				Timeout:          2 * time.Second,
				HealthyThreshold: 1 * time.Second,
				MaxRetries:       -1,
				RetryDelay:       500 * time.Millisecond,
			},
			wantError: true,
		},
		{
			name: "valid config",
			config: Config{
				Timeout:          2 * time.Second,
				HealthyThreshold: 1 * time.Second,
				MaxRetries:       3,
				RetryDelay:       500 * time.Millisecond,
			},
			wantError: false,
		},
		{
			name: "zero timeout",
			config: Config{
				Timeout:          0 * time.Second,
				HealthyThreshold: 1 * time.Second,
				MaxRetries:       3,
				RetryDelay:       500 * time.Millisecond,
			},
			wantError: true,
		},
		{
			name: "zero healthy threshold",
			config: Config{
				Timeout:          2 * time.Second,
				HealthyThreshold: 0 * time.Second,
				MaxRetries:       3,
				RetryDelay:       500 * time.Millisecond,
			},
			wantError: true,
		},
		{
			name: "zero retry delay",
			config: Config{
				Timeout:          2 * time.Second,
				HealthyThreshold: 1 * time.Second,
				MaxRetries:       3,
				RetryDelay:       0 * time.Millisecond,
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			gotError := err != nil

			if test.wantError != gotError {
				t.Errorf(
					"want error = %v, got error = %v", test.wantError, gotError)
			}
		})
	}
}
