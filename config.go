package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Timeout          time.Duration
	HealthyThreshold time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
}

func getDurationEnv(key string, defaultValue time.Duration) (time.Duration, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return duration, nil
}

func getIntEnv(key string, defaultValue int) (int, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue, nil
	}
	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return parsedValue, nil
}

func loadConfig() (Config, error) {
	timeout, err := getDurationEnv("MONITOR_TIMEOUT", 2*time.Second)
	if err != nil {
		return Config{}, err
	}

	health, err := getDurationEnv("MONITOR_HEALTHY_THRESHOLD", 1*time.Second)
	if err != nil {
		return Config{}, err
	}

	retries, err := getIntEnv("MONITOR_MAX_RETRIES", 3)
	if err != nil {
		return Config{}, err
	}

	delay, err := getDurationEnv("MONITOR_RETRY_DELAY", 500*time.Millisecond)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Timeout:          timeout,
		HealthyThreshold: health,
		MaxRetries:       retries,
		RetryDelay:       delay,
	}
	err = config.Validate()
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("MONITOR_MAX_RETRIES cannot be negative: %d", c.MaxRetries)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("MONITOR_TIMEOUT cannot be 0 or less: %s", c.Timeout)
	}

	if c.HealthyThreshold <= 0 {
		return fmt.Errorf(
			"MONITOR_HEALTHY_THRESHOLD must be greater than 0: %s",
			c.HealthyThreshold,
		)
	}

	if c.RetryDelay <= 0 {
		return fmt.Errorf(
			"MONITOR_RETRY_DELAY must be greater than 0: %s",
			c.RetryDelay,
		)
	}

	return nil
}
