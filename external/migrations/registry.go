package migrations

import "fsos-server/internal/domain/ports"

var All []ports.Migration

func Register(m ports.Migration) {
	All = append(All, m)
}
