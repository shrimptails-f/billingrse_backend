package presentation

import "github.com/stretchr/testify/mock"

type MockLogger struct {
	logger mock.Mock
	mock.Mock
}

func (m *MockLogger) Debug(message interface{}, args ...interface{}) {
}

func (m *MockLogger) Info(message string, args ...interface{}) {
}

func (m *MockLogger) Warn(message string, args ...interface{}) {
}

// Error -.
func (m *MockLogger) Error(message interface{}, args ...interface{}) {
}

func (m *MockLogger) Fatal(message interface{}, args ...interface{}) {
}

func (m *MockLogger) log(message string, args ...interface{}) {
}

func (m *MockLogger) msg(level string, message interface{}, args ...interface{}) {
}
