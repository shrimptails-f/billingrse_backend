package di

// import (
// 	"business/internal/app/presentation"
// 	"business/internal/library/gmail"
// 	"business/internal/library/gmailService"
// 	"business/internal/library/logger"
// 	"business/internal/library/mysql"
// 	"business/internal/library/openai"
// 	"business/internal/library/oswrapper"
// 	"business/internal/library/ratelimit"
// 	"business/internal/library/timewrapper"
// 	"context"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// type noopLimiter struct{}

// func (n noopLimiter) Wait(ctx context.Context) error {
// 	return nil
// }

// func TestBuildContainer_NoError(t *testing.T) {
// 	// ダミー（空実装）具象を生成
// 	conn := &mysql.MySQL{}
// 	oa := &openai.Client{}
// 	gs := &gmailService.Client{}
// 	gc := &gmail.Client{}
// 	osw := &oswrapper.OsWrapper{}

// 	log, err := logger.New("info")
// 	assert.NoError(t, err)

// 	provider := ratelimit.NewProvider(nil, timewrapper.NewClock(), osw, log)

// 	container := BuildContainer(conn, oa, gs, gc, osw, provider, log)

// 	// invokeだけを行い、実行はしない（副作用なし）
// 	err = container.Invoke(func(
// 		_ *mysql.MySQL,
// 		_ *openai.Client,
// 		_ *gmailService.Client,
// 		_ *oswrapper.OsWrapper,
// 	) {
// 		// 何もしない
// 	})

// 	assert.NoError(t, err)
// }

// func TestBuildContainer_WithPresentationLayer(t *testing.T) {
// 	// 必須の環境変数を設定
// 	t.Setenv("AGENT_TOKEN_KEY_V1", "this-is-a-32-byte-key-material!!")
// 	t.Setenv("AGENT_TOKEN_SALT", "agent-salt")
// 	t.Setenv("EMAIL_TOKEN_KEY_V1", "this-is-a-32-byte-key-material!!")
// 	t.Setenv("EMAIL_TOKEN_SALT", "email-salt")

// 	// ダミー（空実装）具象を生成
// 	conn := &mysql.MySQL{}
// 	oa := &openai.Client{}
// 	gs := &gmailService.Client{}
// 	gc := &gmail.Client{}
// 	osw := &oswrapper.OsWrapper{}

// 	log, err := logger.New("info")
// 	assert.NoError(t, err)

// 	provider := ratelimit.NewProvider(nil, timewrapper.NewClock(), osw, log)

// 	container := BuildContainer(conn, oa, gs, gc, osw, provider, log)

// 	// presentation層の依存注入をテスト
// 	err = container.Invoke(func(controller *presentation.AnalyzeEmailController) {
// 		// controllerが正常に注入されることを確認
// 		assert.NotNil(t, controller)
// 	})

// 	assert.NoError(t, err)
// }
