package mysql

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type structuredGormLogger struct {
	log           logger.Interface
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

func newGormLogger(log logger.Interface, slowThreshold time.Duration) gormlogger.Interface {
	if log == nil {
		log = logger.NewNop()
	}

	return &structuredGormLogger{
		log:           log.With(logger.Component("mysql")),
		level:         gormlogger.Warn,
		slowThreshold: slowThreshold,
	}
}

func newSilentGormLogger() gormlogger.Interface {
	return &structuredGormLogger{
		log:           logger.NewNop(),
		level:         gormlogger.Silent,
		slowThreshold: time.Second,
	}
}

func newErrorOnlyGormLogger(log logger.Interface, slowThreshold time.Duration) gormlogger.Interface {
	return newGormLogger(log, slowThreshold).LogMode(gormlogger.Error)
}

func (l *structuredGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &structuredGormLogger{
		log:           l.log,
		level:         level,
		slowThreshold: l.slowThreshold,
	}
}

func (l *structuredGormLogger) Info(context.Context, string, ...interface{}) {}

func (l *structuredGormLogger) Warn(context.Context, string, ...interface{}) {}

func (l *structuredGormLogger) Error(context.Context, string, ...interface{}) {}

func (l *structuredGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	logErr := err != nil && !errors.Is(err, gorm.ErrRecordNotFound)
	slow := l.slowThreshold > 0 && elapsed > l.slowThreshold
	if !logErr && !slow {
		return
	}

	sql, rowsAffected := fc()
	fields := []logger.Field{
		logger.String("db_system", "mysql"),
		logger.String("db_operation", sqlOperation(sql)),
		logger.Int("latency_ms", int(elapsed.Milliseconds())),
	}
	if rowsAffected >= 0 {
		fields = append(fields, logger.Int("rows_affected", int(rowsAffected)))
	}

	reqLog := l.log
	if withContext, withContextErr := l.log.WithContext(ctx); withContextErr == nil {
		reqLog = withContext
	}

	switch {
	case logErr && l.level >= gormlogger.Error:
		reqLog.Error("db_query_failed", append(fields, logger.Err(err))...)
	case slow && l.level >= gormlogger.Warn:
		reqLog.Warn("db_query_slow", fields...)
	}
}

func sqlOperation(sql string) string {
	normalized := strings.TrimSpace(sql)
	if normalized == "" {
		return "unknown"
	}

	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return "unknown"
	}

	return strings.ToLower(parts[0])
}

var _ gormlogger.Interface = (*structuredGormLogger)(nil)

func (l *structuredGormLogger) String() string {
	return fmt.Sprintf("structuredGormLogger(level=%d)", l.level)
}
