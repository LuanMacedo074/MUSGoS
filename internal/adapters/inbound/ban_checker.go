package inbound

import (
	"time"

	"fsos-server/internal/domain/ports"
)

type BanChecker struct {
	db    ports.DBAdapter
	cache ports.Cache
}

func NewBanChecker(db ports.DBAdapter, cache ports.Cache) *BanChecker {
	return &BanChecker{db: db, cache: cache}
}

func (b *BanChecker) IsIPBanned(host string) bool {
	if b.db == nil {
		return false
	}

	cacheKey := "ban:ip:" + host

	if b.cache != nil {
		if val, err := b.cache.Get(cacheKey); err == nil && val != nil {
			return string(val) == "1"
		}
	}

	ban, err := b.db.GetActiveBanByIP(host)
	banned := err == nil && ban != nil

	if b.cache != nil {
		val := "0"
		if banned {
			val = "1"
		}
		b.cache.Set(cacheKey, []byte(val), 30*time.Second)
	}

	return banned
}
