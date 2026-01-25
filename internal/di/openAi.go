package di

// import (
// 	"business/internal/library/logger"
// 	"business/internal/library/openai"
// 	"business/internal/library/oswrapper"
// 	aiadapter "business/internal/messaging/infrastructure/ai/openai/adapter"

// 	"go.uber.org/dig"
// )

// // ProvideOpenAiDependencies OpenAi APIを実行する機能群の依存注入設定
// func ProvideOpenAiDependencies(container *dig.Container) {
// 	_ = container.Provide(func(oa *openai.Client, osw *oswrapper.OsWrapper, log logger.Interface) *aiadapter.AnalyzerAdapter {
// 		return aiadapter.NewAnalyzerAdapter(oa, osw, log)
// 	})
// }
