package mocklibrary

import "fmt"

// OsWrapperMock provides a reusable implementation of oswrapper.OsWapperInterface
// for tests that need to control environment variables and file reads.
type OsWrapperMock struct {
	env         map[string]string
	presentEnv  map[string]struct{}
	files       map[string]string
	readFileErr error
}

// NewOsWrapperMock returns a mock initialized with the provided environment variables.
func NewOsWrapperMock(env map[string]string) *OsWrapperMock {
	mock := &OsWrapperMock{
		env:        map[string]string{},
		presentEnv: map[string]struct{}{},
		files:      map[string]string{},
	}
	return mock.WithEnv(env)
}

// WithEnv merges the provided key/value pairs into the mock environment.
func (m *OsWrapperMock) WithEnv(env map[string]string) *OsWrapperMock {
	if m == nil {
		return m
	}
	if env == nil {
		return m
	}
	for k, v := range env {
		m.env[k] = v
		if v == "" {
			delete(m.presentEnv, k)
			continue
		}
		m.presentEnv[k] = struct{}{}
	}
	return m
}

// WithEnvValue registers a key as explicitly present, even when the value is empty.
func (m *OsWrapperMock) WithEnvValue(key, value string) *OsWrapperMock {
	if m == nil {
		return m
	}
	m.env[key] = value
	m.presentEnv[key] = struct{}{}
	return m
}

// WithFile registers mock file contents for ReadFile calls.
func (m *OsWrapperMock) WithFile(path, contents string) *OsWrapperMock {
	if m == nil {
		return m
	}
	m.files[path] = contents
	return m
}

// WithReadFileError configures ReadFile to always return the provided error.
func (m *OsWrapperMock) WithReadFileError(err error) *OsWrapperMock {
	if m == nil {
		return m
	}
	m.readFileErr = err
	return m
}

// ReadFile returns the registered file contents or the configured error.
func (m *OsWrapperMock) ReadFile(path string) (string, error) {
	if m.readFileErr != nil {
		return "", m.readFileErr
	}
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return "", nil
}

// GetEnv looks up a value from the mock environment.
func (m *OsWrapperMock) GetEnv(key string) (string, error) {
	if m == nil {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	if _, ok := m.presentEnv[key]; ok {
		return m.env[key], nil
	}
	if value, ok := m.env[key]; ok && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}
