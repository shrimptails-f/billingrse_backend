package di

import (
	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	"business/internal/auth/application"
	"business/internal/library/gmail"
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/openai"
	"business/internal/library/oswrapper"
	"business/internal/library/ratelimit"
	"business/internal/library/timewrapper"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
)

type limiterDeps struct {
	dig.In

	GmailLimiter  ratelimit.Limiter `name:"gmailLimiter"`
	OpenAILimiter ratelimit.Limiter `name:"openaiLimiter"`
}

func newBuildContainerTestDeps() (*mysql.MySQL, *openai.Client, *gmailService.Client, *gmail.Client, *oswrapper.OsWrapper, *ratelimit.Provider, logger.Interface) {
	conn := &mysql.MySQL{}
	oa := &openai.Client{}
	gs := &gmailService.Client{}
	gc := &gmail.Client{}
	osw := &oswrapper.OsWrapper{}
	log := logger.NewNop()
	provider := ratelimit.NewProvider(nil, timewrapper.NewClock(), osw, log)

	return conn, oa, gs, gc, osw, provider, log
}

func TestProvideCommonDependencies_RegistersDependencies(t *testing.T) {
	t.Parallel()

	conn, oa, gs, gc, osw, provider, log := newBuildContainerTestDeps()
	container := dig.New()

	ProvideCommonDependencies(container, conn, oa, gs, gc, osw, provider, log)

	err := container.Invoke(func(
		gotConn *mysql.MySQL,
		gotOA *openai.Client,
		gotGS *gmailService.Client,
		gotGC *gmail.Client,
		gotOSW *oswrapper.OsWrapper,
		gotOSWInterface oswrapper.OsWapperInterface,
		gotProvider *ratelimit.Provider,
		gotLog logger.Interface,
		gotClock timewrapper.ClockInterface,
		limiters limiterDeps,
	) {
		require.NotNil(t, gotOSWInterface)

		resolvedOSW, ok := gotOSWInterface.(*oswrapper.OsWrapper)
		require.True(t, ok)

		assert.Same(t, conn, gotConn)
		assert.Same(t, oa, gotOA)
		assert.Same(t, gs, gotGS)
		assert.Same(t, gc, gotGC)
		assert.Same(t, osw, gotOSW)
		assert.Same(t, osw, resolvedOSW)
		assert.Same(t, provider, gotProvider)
		assert.Same(t, log, gotLog)
		assert.NotNil(t, gotClock)
		assert.NotNil(t, limiters.GmailLimiter)
		assert.NotNil(t, limiters.OpenAILimiter)
	})

	require.NoError(t, err)
}

func TestBuildContainer_ResolvesAuthPresentation(t *testing.T) {
	t.Parallel()

	conn, oa, gs, gc, osw, provider, log := newBuildContainerTestDeps()
	container := BuildContainer(conn, oa, gs, gc, osw, provider, log)

	err := container.Invoke(func(
		controller *authpresentation.Controller,
		authMiddleware *middleware.AuthMiddleware,
		usecase application.AuthUseCaseInterface,
	) {
		assert.NotNil(t, controller)
		assert.NotNil(t, authMiddleware)
		assert.NotNil(t, usecase)
	})

	require.NoError(t, err)
}
