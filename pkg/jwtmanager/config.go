package jwtmanager

import "time"

type Config struct {
	PrivateKeyFile       string
	PublicKeyFile        string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}
