package di

import (
	"testing"

	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	"business/internal/auth/application"
	"business/internal/library/crypto"
	"business/internal/library/gmail"
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/openai"
	"business/internal/library/oswrapper"
	"business/internal/library/ratelimit"
	"business/internal/library/timewrapper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
	"golang.org/x/crypto/bcrypt"
)

func newBuildContainerTestDeps() (*mysql.MySQL, *openai.Client, *gmailService.Client, *gmail.Client, *oswrapper.OsWrapper, *ratelimit.Provider, *logger.Logger, *crypto.Vault) {
	conn := &mysql.MySQL{}
	oa := &openai.Client{}
	gs := &gmailService.Client{}
	gc := &gmail.Client{}
	osw := &oswrapper.OsWrapper{}
	log := logger.NewNop()
	provider := ratelimit.NewProvider(nil, timewrapper.NewClock(), osw, log)
	vault, _ := crypto.NewVault(crypto.VaultConfig{
		KeyMaterial: []byte("01234567890123456789012345678901"),
		Salt:        []byte("test-salt-value"),
		Info:        "email-credential-encryption",
		BcryptCost:  bcrypt.MinCost,
	})

	return conn, oa, gs, gc, osw, provider, log, vault
}

func TestProvideCommonDependencies_RegistersDependencies(t *testing.T) {
	t.Parallel()

	conn, oa, gs, gc, osw, provider, log, vault := newBuildContainerTestDeps()
	container := dig.New()

	ProvideCommonDependencies(container, conn, oa, gs, gc, osw, provider, log, vault)

	err := container.Invoke(func(
		gotConn *mysql.MySQL,
		gotOA *openai.Client,
		gotGS *gmailService.Client,
		gotGC *gmail.Client,
		gotOSW *oswrapper.OsWrapper,
		gotProvider *ratelimit.Provider,
		gotLog *logger.Logger,
		gotClock *timewrapper.Clock,
	) {
		assert.Same(t, conn, gotConn)
		assert.Same(t, oa, gotOA)
		assert.Same(t, gs, gotGS)
		assert.Same(t, gc, gotGC)
		assert.Same(t, osw, gotOSW)
		assert.Same(t, provider, gotProvider)
		assert.Same(t, log, gotLog)
		assert.NotNil(t, gotClock)
	})

	require.NoError(t, err)
}

func TestProvideCommonDependencies_DoesNotRegisterLoggerAndOSWrapperInterfaceAliases(t *testing.T) {
	t.Parallel()

	conn, oa, gs, gc, osw, provider, log, vault := newBuildContainerTestDeps()
	container := dig.New()

	ProvideCommonDependencies(container, conn, oa, gs, gc, osw, provider, log, vault)

	err := container.Invoke(func(logger.Interface) {})
	require.Error(t, err)

	err = container.Invoke(func(oswrapper.OsWapperInterface) {})
	require.Error(t, err)
}

func TestProvideCommonDependencies_RegistersClockInterfaceAlias(t *testing.T) {
	t.Parallel()

	conn, oa, gs, gc, osw, provider, log, vault := newBuildContainerTestDeps()
	container := dig.New()

	ProvideCommonDependencies(container, conn, oa, gs, gc, osw, provider, log, vault)

	err := container.Invoke(func(timewrapper.ClockInterface) {})
	require.NoError(t, err)
}

func TestBuildContainer_ResolvesAuthPresentation(t *testing.T) {
	t.Parallel()

	conn, oa, gs, gc, osw, provider, log, vault := newBuildContainerTestDeps()
	container := BuildContainer(conn, oa, gs, gc, osw, provider, log, vault)

	err := container.Invoke(func(
		controller *authpresentation.Controller,
		authMiddleware *middleware.AuthMiddleware,
		usecase *application.AuthUseCase,
	) {
		assert.NotNil(t, controller)
		assert.NotNil(t, authMiddleware)
		assert.NotNil(t, usecase)
	})

	require.NoError(t, err)
}

func TestBuildContainer_DoesNotResolveUseCaseInterface(t *testing.T) {
	t.Parallel()

	conn, oa, gs, gc, osw, provider, log, vault := newBuildContainerTestDeps()
	container := BuildContainer(conn, oa, gs, gc, osw, provider, log, vault)

	err := container.Invoke(func(application.AuthUseCaseInterface) {})
	require.Error(t, err)
}
