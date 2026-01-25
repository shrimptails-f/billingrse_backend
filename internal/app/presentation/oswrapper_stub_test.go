package presentation

import (
	"fmt"

	"business/internal/auth/application"
	"business/internal/library/logger"
	"business/internal/library/oswrapper"
)

type stubOsWrapper struct {
	vars  map[string]string
	files map[string]string
}

var _ oswrapper.OsWapperInterface = (*stubOsWrapper)(nil)

func (s *stubOsWrapper) ReadFile(path string) (string, error) {
	if s.files == nil {
		return "", fmt.Errorf("file %s not configured", path)
	}
	if data, ok := s.files[path]; ok {
		return data, nil
	}
	return "", fmt.Errorf("file %s not configured", path)
}

func (s *stubOsWrapper) GetEnv(key string) (string, error) {
	if v, ok := s.vars[key]; ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}

func newStubOsWrapper(vars map[string]string) oswrapper.OsWapperInterface {
	base := map[string]string{
		"APP":    "local",
		"DOMAIN": "localhost",
	}
	for k, v := range vars {
		base[k] = v
	}
	return &stubOsWrapper{vars: base}
}

// newTestAuthController wires AuthController with a stub os wrapper.
func newTestAuthController(usecase application.AuthUseCaseInterface, log logger.Interface) *AuthController {
	return NewAuthController(usecase, log, newStubOsWrapper(nil))
}

func newTestAuthControllerWithVars(usecase application.AuthUseCaseInterface, log logger.Interface, vars map[string]string) *AuthController {
	return NewAuthController(usecase, log, newStubOsWrapper(vars))
}
