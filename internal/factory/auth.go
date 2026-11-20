package factory

import (
	"musgos/internal/adapters/outbound"
	"musgos/internal/domain/ports"
)

// AuthConfig supplies application schema and compatibility choices independently of transport settings.
type AuthConfig struct {
	Refresh           outbound.RefreshTokenConfig
	LegacyUserIDClaim string
}

type AuthStores struct {
	Tokens  ports.TokenStore
	Refresh ports.RefreshTokenStore
}

func NewAuthStores(cache ports.Cache, qb ports.QueryBuilder, config AuthConfig) AuthStores {
	stores := AuthStores{}
	if cache != nil {
		stores.Tokens = outbound.NewCacheTokenStore(cache, outbound.DefaultTokenTTL).WithLegacyUserIDClaim(config.LegacyUserIDClaim)
	}
	if qb != nil {
		stores.Refresh = outbound.NewRefreshTokenStore(qb, nil, config.Refresh)
	}
	return stores
}
