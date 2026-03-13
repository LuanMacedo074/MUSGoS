package factory

import (
	"fmt"

	"fsos-server/internal/adapters/outbound"
	"fsos-server/internal/domain/ports"
)

const smusDefaultKey = "IPAddress resolution"

func NewCipher(cipherType, key string) (ports.Cipher, error) {
	switch cipherType {
	case "blowfish":
		// SMUS protocol: keys shorter than 20 chars get padded with the default key
		if len(key) < 20 {
			key = key + smusDefaultKey
		}
		bf := outbound.NewBlowfish(key)
		bf.SetKey()
		return bf, nil
	default:
		return nil, fmt.Errorf("unsupported cipher type: %s", cipherType)
	}
}
