package redisclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"business/internal/library/logger"
	"business/internal/library/oswrapper"
	"business/internal/library/redis/script"

	goredis "github.com/redis/go-redis/v9"
	maintnotifications "github.com/redis/go-redis/v9/maintnotifications"
)

// Config represents Redis connection settings.
type Config struct {
	URL string
}

var (
	defaultOsWrapper   oswrapper.OsWapperInterface = oswrapper.New(nil)
	defaultOsWrapperMu sync.RWMutex
)

// SetDefaultOsWrapper overrides the package-level os wrapper used when none is provided.
// Passing nil resets the wrapper to the real OS implementation.
func SetDefaultOsWrapper(osw oswrapper.OsWapperInterface) {
	defaultOsWrapperMu.Lock()
	defer defaultOsWrapperMu.Unlock()
	if osw == nil {
		defaultOsWrapper = oswrapper.New(nil)
		return
	}
	defaultOsWrapper = osw
}

func currentOsWrapper() oswrapper.OsWapperInterface {
	defaultOsWrapperMu.RLock()
	defer defaultOsWrapperMu.RUnlock()
	return defaultOsWrapper
}

// Client wraps a Redis client and implements ClientInterface.
type Client struct {
	client          *goredis.Client
	scriptCache     map[string]string
	scriptCacheMu   sync.Mutex
	rateLimitScript script.Script
	log             logger.Interface
}

// EvalScript executes a Lua script with automatic recovery from NOSCRIPT errors.
// If the script is not loaded in Redis, it will be loaded automatically and retried.
func (c *Client) EvalScript(ctx context.Context, scr script.Script, keys []string, args ...interface{}) (interface{}, error) {
	// Check cache first
	c.scriptCacheMu.Lock()
	cachedSHA, hasCached := c.scriptCache[scr.Name]
	c.scriptCacheMu.Unlock()

	// Determine which SHA to use
	shaToUse := scr.SHA
	if hasCached {
		shaToUse = cachedSHA
	}

	// First attempt: try EvalSha with the determined SHA
	result, err := c.client.EvalSha(ctx, shaToUse, keys, args...).Result()
	if err == nil {
		// Success: ensure this SHA is cached for future use
		if !hasCached {
			c.scriptCacheMu.Lock()
			if c.scriptCache == nil {
				c.scriptCache = make(map[string]string)
			}
			c.scriptCache[scr.Name] = shaToUse
			c.scriptCacheMu.Unlock()
		}
		return result, nil
	}

	// Check if the error is NOSCRIPT
	errStr := err.Error()
	if !strings.Contains(errStr, "NOSCRIPT") {
		return nil, err
	}

	// NOSCRIPT detected: load the script and retry
	c.scriptCacheMu.Lock()
	defer c.scriptCacheMu.Unlock()

	// Double-check: another goroutine might have already loaded it
	cachedSHA, hasCached = c.scriptCache[scr.Name]
	if hasCached {
		result, err = c.client.EvalSha(ctx, cachedSHA, keys, args...).Result()
		if err == nil {
			return result, nil
		}
		// If still NOSCRIPT, continue to load
		if !strings.Contains(err.Error(), "NOSCRIPT") {
			return nil, err
		}
	}

	// Load the script and get the SHA
	loadedSHA, loadErr := c.client.ScriptLoad(ctx, scr.Body).Result()
	if loadErr != nil {
		return nil, fmt.Errorf("failed to load script %s: %w", scr.Name, loadErr)
	}

	// Cache the loaded SHA
	if c.scriptCache == nil {
		c.scriptCache = make(map[string]string)
	}
	c.scriptCache[scr.Name] = loadedSHA

	// Retry with the loaded SHA
	result, err = c.client.EvalSha(ctx, loadedSHA, keys, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("script execution failed after load for %s: %w", scr.Name, err)
	}

	return result, nil
}

// EvalSha executes a Lua script identified by its SHA1 hash and returns its result.
func (c *Client) EvalSha(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error) {
	return c.client.EvalSha(ctx, sha, keys, args...).Result()
}

// RunRateLimitScript executes the rate limit Lua script and returns parsed results.
// It handles all Redis-specific concerns: script execution, result parsing, error logging,
// and wrapping errors as ErrRedisUnavailable when appropriate.
func (c *Client) RunRateLimitScript(ctx context.Context, params RateLimitParams) (RateLimitResult, error) {
	log := c.log
	if log == nil {
		log = logger.NewNop()
	}

	// Build args for Lua script
	nowUnix := params.Time.Unix()
	args := []interface{}{
		params.Namespace,
		params.Bucket,
		nowUnix,
		len(params.Windows),
	}

	for _, w := range params.Windows {
		args = append(args, w.SizeSeconds, w.Limit)
	}

	// Execute script
	result, err := c.EvalScript(ctx, c.rateLimitScript, []string{}, args...)
	if err != nil {
		addr, db := c.redisAddrAndDB()
		log.Error("rate limiter redis error",
			logger.String("namespace", params.Namespace),
			logger.String("addr", addr),
			logger.Int("db", db),
			logger.Err(err))
		return RateLimitResult{}, &ErrRedisUnavailable{
			Err: fmt.Errorf(
				"redis script execution failed (namespace=%s addr=%s db=%d): %w",
				params.Namespace,
				addr,
				db,
				err,
			),
		}
	}

	// Parse result
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 4 {
		err := fmt.Errorf("unexpected redis response format for namespace=%s", params.Namespace)
		log.Error("rate limiter redis response error",
			logger.String("namespace", params.Namespace),
			logger.Err(err))
		return RateLimitResult{}, &ErrRedisUnavailable{
			Err: fmt.Errorf("unexpected redis response format for namespace=%s", params.Namespace),
		}
	}

	allowed := parseInt64(resultSlice[0])
	windowSize := parseInt64(resultSlice[1])
	limit := parseInt64(resultSlice[2])
	total := parseInt64(resultSlice[3])

	return RateLimitResult{
		Allowed:       allowed == 1,
		WindowSeconds: int(windowSize),
		Limit:         int(limit),
		Current:       int(total),
	}, nil
}

func (c *Client) redisAddrAndDB() (string, int) {
	if c.client == nil {
		return "", 0
	}
	opts := c.client.Options()
	if opts == nil {
		return "", 0
	}
	return opts.Addr, opts.DB
}

// parseInt64 converts redis script outputs to int64, tolerating strings and ints.
func parseInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		return 0
	}
}

// New creates a new redis client using the provided configuration.
func New(cfg Config, osw oswrapper.OsWapperInterface, log logger.Interface) (ClientInterface, error) {
	if osw == nil {
		osw = currentOsWrapper()
	}
	if log == nil {
		log = logger.NewNop()
	}
	log = log.With(logger.String("component", "redis_client"))

	url := cfg.URL
	if url == "" {
		var err error
		url, err = buildURLFromComponents(osw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL: %w", err)
		}
	}

	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	disableMaintNotifications(opts)

	// Load rate limit script from file
	rateLimitScript, err := script.New(osw, "rate_limit", "")
	if err != nil {
		return nil, err
	}

	return &Client{
		client:          goredis.NewClient(opts),
		scriptCache:     make(map[string]string),
		rateLimitScript: rateLimitScript,
		log:             log,
	}, nil
}

func disableMaintNotifications(opts *goredis.Options) {
	if opts == nil {
		return
	}
	if opts.MaintNotificationsConfig == nil {
		opts.MaintNotificationsConfig = &maintnotifications.Config{}
	}
	opts.MaintNotificationsConfig.Mode = maintnotifications.ModeDisabled
}

func buildURLFromComponents(osw oswrapper.OsWapperInterface) (string, error) {
	host, err := osw.GetEnv("REDIS_HOST")
	if err != nil {
		return "", err
	}
	port, err := osw.GetEnv("REDIS_PORT")
	if err != nil {
		return "", err
	}
	password, err := osw.GetEnv("REDIS_PASSWORD")
	if err != nil {
		return "", err
	}
	db, err := osw.GetEnv("REDIS_DB")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("redis://:%s@%s:%s/%s", password, host, port, db), nil
}
