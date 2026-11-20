package migrations

import "musgos/internal/domain/ports"

var All []ports.Migration

func Register(m ports.Migration) {
	All = append(All, m)
}
