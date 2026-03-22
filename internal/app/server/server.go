package server

import (
	"business/internal/app/middleware"
	v1 "business/internal/app/router"
	"business/internal/di"
	"business/internal/library/crypto"
	"business/internal/library/gmail"
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/openai"
	"business/internal/library/oswrapper"
	"business/internal/library/ratelimit"
	"business/internal/library/secret"
	"business/internal/library/timewrapper"

	"context"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
)

func Run() {
	g := gin.New()
	clock := timewrapper.NewClock()

	ctx := context.Background()
	secretClient, err := secret.New(ctx)
	if err != nil {
		panic("シークレットクライアント初期化に失敗しました: " + err.Error())
	}
	osw := oswrapper.New(secretClient)
	environment := strings.TrimSpace(os.Getenv("APP"))
	if environment == "" {
		panic("初期化に失敗しました: os.GetenvでAPP項目を取得できませんでした。")
	}

	baseLogger, err := logger.New("info", "backend", environment)
	if err != nil {
		panic("ロガー初期化に失敗しました: " + err.Error())
	}
	defer baseLogger.Sync()

	serverLogger := baseLogger.With(logger.Component("server"))
	routerLogger := baseLogger.With(logger.Component("router"))

	// DBインスタンス生成
	db, err := mysql.New(osw, baseLogger)
	if err != nil {
		serverLogger.Error("DB 初期化時にエラーが発生しました", logger.Err(err))
		return
	}

	providerLogger := baseLogger.With(logger.Component("ratelimit_provider"))
	provider, err := ratelimit.NewProviderFromEnv(osw, providerLogger)
	if err != nil {
		serverLogger.Error("レートリミット初期化時にエラーが発生しました", logger.Err(err))
		return
	}
	gmailLimiter := provider.GetGmailLimiter()
	openaiLimiter := provider.GetOpenAILimiter()

	// OpenAiクライアント作成
	apiKey, err := osw.GetEnv("OPENAI_API_KEY")
	if err != nil {
		serverLogger.Error("環境変数 OPENAI_API_KEY の取得に失敗しました", logger.Err(err))
		return
	}
	oa := openai.New(apiKey, openaiLimiter, baseLogger)
	gs := gmailService.New()
	gc := gmail.New(gmailLimiter, baseLogger)
	emailTokenKey, err := osw.GetEnv("EMAIL_TOKEN_KEY_V1")
	if err != nil {
		serverLogger.Error("failed to read EMAIL_TOKEN_KEY_V1", logger.Err(err))
		return
	}

	emailTokenSalt, err := osw.GetEnv("EMAIL_TOKEN_SALT")
	if err != nil {
		serverLogger.Error("failed to read EMAIL_TOKEN_SALT", logger.Err(err))
		return
	}

	// Vaultインスタンス生成
	vault, err := crypto.NewVault(crypto.VaultConfig{
		KeyMaterial: []byte(emailTokenKey),
		Salt:        []byte(emailTokenSalt),
		Info:        "email-credential-encryption",
		BcryptCost:  bcrypt.DefaultCost,
	})
	if err != nil {
		serverLogger.Error("Vault 初期化時にエラーが発生しました", logger.Err(err))
		return
	}

	// DIを行う
	container := di.BuildContainer(db, oa, gs, gc, osw, provider, baseLogger, vault)
	var isUseSSL string
	isUseSSL, err = osw.GetEnv("USE_SSL")
	if err != nil {
		serverLogger.Error("環境変数 USE_SSL の取得に失敗しました", logger.Err(err))
		return
	}

	var frontDmain string
	if isUseSSL == "TRUE" || isUseSSL == "true" {
		frontDmain, err = osw.GetEnv("FRONT_SSL_DOMAIN")
		if err != nil {
			serverLogger.Error("環境変数 FRONT_SSL_DOMAIN の取得に失敗しました", logger.Err(err))
			return
		}
	} else {
		frontDmain, err = osw.GetEnv("FRONT_DOMAIN")
		if err != nil {
			serverLogger.Error("環境変数 FRONT_DOMAIN の取得に失敗しました", logger.Err(err))
			return
		}

	}

	g.Use(middleware.RequestID())
	g.Use(middleware.RequestSummary(baseLogger, clock))
	g.Use(middleware.Recovery(baseLogger))
	g.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", frontDmain)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
	router, err := v1.Router(g, container, routerLogger, frontDmain)
	if err != nil {
		routerLogger.Error("failed to start router", logger.Err(err))
		return
	}
	addr := ":8080"
	serverLogger.Info("HTTP サーバーを起動します", logger.String("addr", addr))
	router.Run(addr)
}
