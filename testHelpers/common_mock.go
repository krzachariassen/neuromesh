package testHelpers

import (
	"github.com/stretchr/testify/mock"
)

// MockLogger provides a testify-based mock for logger operations
type MockLogger struct {
	mock.Mock
}

// NewMockLogger creates a new mock logger instance
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	args := []interface{}{msg}
	args = append(args, fields...)
	m.Called(args...)
}

func (m *MockLogger) Error(msg string, err error, fields ...interface{}) {
	args := []interface{}{msg, err}
	args = append(args, fields...)
	m.Called(args...)
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	args := []interface{}{msg}
	args = append(args, fields...)
	m.Called(args...)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	args := []interface{}{msg}
	args = append(args, fields...)
	m.Called(args...)
}
