package crypto

// MUSBlowfish is a wrapper around MUSBlowfishCipher that handles automatic reset
type MUSBlowfish struct {
    cipher *MUSBlowfishCipher
}

// NewMUSBlowfish creates a new MUS Blowfish wrapper with a string key
func NewMUSBlowfish(key string) *MUSBlowfish {
    // Prepare key according to SMUS rules
    preparedKey := prepareKey(key)
    
    return &MUSBlowfish{
        cipher: NewMUSBlowfishCipher([]byte(preparedKey)),
    }
}

// NewMUSBlowfishFromBytes creates a new MUS Blowfish wrapper with a byte array key
func NewMUSBlowfishFromBytes(key []byte) *MUSBlowfish {
    return &MUSBlowfish{
        cipher: NewMUSBlowfishCipher(key),
    }
}

// NewMUSBlowfishDefault creates a new MUS Blowfish wrapper using pre-calculated boxes
func NewMUSBlowfishDefault() *MUSBlowfish {
    return &MUSBlowfish{
        cipher: NewMUSBlowfishCipherPreCalc(),
    }
}

// Encode encrypts data in place WITHOUT resetting the cipher
func (m *MUSBlowfish) Encode(data []byte) {
    m.cipher.Encrypt(data)
    // NO RESET - maintain state between messages
}

// Decode decrypts data in place WITHOUT resetting the cipher
func (m *MUSBlowfish) Decode(data []byte) {
    m.cipher.Decrypt(data)
    // NO RESET - maintain state between messages
}

// DecodeWithReset decrypts data in place and resets the cipher
func (m *MUSBlowfish) DecodeWithReset(data []byte) {
    m.cipher.Decrypt(data)
    m.cipher.Reset()
}

// Reset manually resets the cipher
func (m *MUSBlowfish) Reset() {
    m.cipher.Reset()
}

// Helper function to handle key concatenation for short keys
func prepareKey(key string) string {
    // If key is less than 20 characters, append default key
    if len(key) < 20 {
        return key + "IPAddress resolution"
    }
    return key
}

// Special key that disables encryption
const NoEncryptionTag = "#NoEncryption"

// Special prefix for encrypting all messages
const EncryptAllMsgsPrefix = "#All"

// IsEncryptionDisabled checks if the key disables encryption
func IsEncryptionDisabled(key string) bool {
    return key == NoEncryptionTag
}