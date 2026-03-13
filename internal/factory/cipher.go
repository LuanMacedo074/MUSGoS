package factory

import (
	"fmt"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

func NewCipher(cipherType, key string) (ports.Cipher, error) {
	switch cipherType {
	case "blowfish":
		bf := outbound.NewBlowfish(key)
		bf.SetKey()
		return bf, nil
	default:
		return nil, fmt.Errorf("unsupported cipher type: %s", cipherType)
	}
}
