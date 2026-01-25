package script

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"business/internal/library/oswrapper"
)

// Script represents a Lua script with its name, body, and SHA1 hash.
type Script struct {
	Name string
	Body string
	SHA  string
}

// New loads a script from the specified path using the provided os wrapper.
// The SHA is computed from the file contents using SHA1.
// If an environment variable SCRIPT_SHA_{NAME} exists, it is used as a fallback.
// When path is empty, the function attempts to read RATE_LIMIT_SCRIPT_PATH from the environment.
func New(osw oswrapper.OsWapperInterface, name, path string) (Script, error) {
	if osw == nil {
		osw = oswrapper.New(nil)
	}

	if path == "" {
		var err error
		path, err = osw.GetEnv("RATE_LIMIT_SCRIPT_PATH")
		if err != nil {
			return Script{}, fmt.Errorf("failed to resolve path for %s: %w", name, err)
		}
	}

	body, err := osw.ReadFile(path)
	if err != nil {
		return Script{}, fmt.Errorf("failed to read script file %s: %w", path, err)
	}

	hash := sha1.New()
	hash.Write([]byte(body))
	sha := hex.EncodeToString(hash.Sum(nil))

	envKey := fmt.Sprintf("SCRIPT_SHA_%s", name)
	if envSHA, err := osw.GetEnv(envKey); err == nil && envSHA != "" {
		sha = strings.TrimSpace(envSHA)
	}

	result := Script{
		Name: name,
		Body: body,
		SHA:  sha,
	}

	return result, nil
}
