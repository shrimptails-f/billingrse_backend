package di

import (
	"business/internal/library/gmail"
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/openai"
	"business/internal/library/oswrapper"
	"business/internal/library/ratelimit"
	"business/internal/library/timewrapper"

	"go.uber.org/dig"
	"gorm.io/gorm"
)

// ProvideCommonDependencies 共通の依存性（例：データベース接続など）を設定する関数
func ProvideCommonDependencies(container *dig.Container, conn *mysql.MySQL, oa *openai.Client, gs *gmailService.Client, gc *gmail.Client, osw *oswrapper.OsWrapper, provider *ratelimit.Provider, log logger.Interface) {
	_ = container.Provide(func() *mysql.MySQL {
		return conn
	})

	_ = container.Provide(func() *openai.Client {
		return oa
	})

	_ = container.Provide(func() *gmailService.Client {
		return gs
	})

	_ = container.Provide(func() *gmail.Client {
		return gc
	})

	_ = container.Provide(func() *oswrapper.OsWrapper {
		return osw
	})

	_ = container.Provide(func(osw *oswrapper.OsWrapper) oswrapper.OsWapperInterface {
		return osw
	})

	// Rate limit provider
	_ = container.Provide(func() *ratelimit.Provider {
		return provider
	})

	_ = container.Provide(func() logger.Interface {
		return log
	})

	// Gmail limiter (from provider)
	_ = container.Provide(func(p *ratelimit.Provider) ratelimit.Limiter {
		return p.GetGmailLimiter()
	}, dig.Name("gmailLimiter"))

	// OpenAI limiter (from provider)
	_ = container.Provide(func(p *ratelimit.Provider) ratelimit.Limiter {
		return p.GetOpenAILimiter()
	}, dig.Name("openaiLimiter"))

	// gorm.DB を提供 (agent などで必要)
	_ = container.Provide(func() *gorm.DB {
		return conn.DB
	})

	// ClockInterface を提供
	_ = container.Provide(func() timewrapper.ClockInterface {
		return timewrapper.NewClock()
	})

	// var wt ct.CustomTime
	// _ = container.Provide(func() ct.WrapperTime {
	// 	return wt
	// })
}

// BuildContainer すべての依存性を統合して設定するコンテナビルダー関数
func BuildContainer(conn *mysql.MySQL, oa *openai.Client, gs *gmailService.Client, gc *gmail.Client, osw *oswrapper.OsWrapper, provider *ratelimit.Provider, log logger.Interface) *dig.Container {
	container := dig.New()

	// 共通の依存性を登録
	ProvideCommonDependencies(container, conn, oa, gs, gc, osw, provider, log)

	// 各機能群の依存性を登録
	ProvideAuthDependencies(container)
	// ProvideEmailAnalysisDependencies(container) // Use messaging abstraction instead
	ProvidePresentationDependencies(container)

	return container
}
