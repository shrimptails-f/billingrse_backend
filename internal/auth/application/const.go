package application

import "time"

const (
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
	defaultTokenTTL        = defaultAccessTokenTTL
)
