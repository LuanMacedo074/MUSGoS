package crypto

import (
    "encoding/binary"
)

const (
    PBOX_ENTRIES = 18
    SBOX_ENTRIES = 256
)

// MUSBlowfishCipher implements the non-standard Blowfish variant used by SMUS
type MUSBlowfishCipher struct {
    pbox  [PBOX_ENTRIES]uint32
    sbox1 [SBOX_ENTRIES]uint32
    sbox2 [SBOX_ENTRIES]uint32
    sbox3 [SBOX_ENTRIES]uint32
    sbox4 [SBOX_ENTRIES]uint32
    cbcIV uint64
}

// Pre-calculated boxes for performance (initialized with default key)
var (
    precalcPbox  [PBOX_ENTRIES]uint32
    precalcSbox1 [SBOX_ENTRIES]uint32
    precalcSbox2 [SBOX_ENTRIES]uint32
    precalcSbox3 [SBOX_ENTRIES]uint32
    precalcSbox4 [SBOX_ENTRIES]uint32
)

// NewMUSBlowfishCipher creates a new cipher with the given key
func NewMUSBlowfishCipher(key []byte) *MUSBlowfishCipher {
    c := &MUSBlowfishCipher{}
    
    // Copy initial P-box
    copy(c.pbox[:], pboxInit[:])
    
    // Copy initial S-boxes
    copy(c.sbox1[:], sboxInit1[:])
    copy(c.sbox2[:], sboxInit2[:])
    copy(c.sbox3[:], sboxInit3[:])
    copy(c.sbox4[:], sboxInit4[:])
    
    // XOR key with P-box
    keyLen := len(key)
    if keyLen > 0 {
        keyPos := 0
        for i := 0; i < PBOX_ENTRIES; i++ {
            var build uint32
            for j := 0; j < 4; j++ {
                build = (build << 8) | uint32(key[keyPos]&0xff)
                keyPos++
                if keyPos == keyLen {
                    keyPos = 0
                }
            }
            c.pbox[i] ^= build
        }
        
        // Encrypt P-box with itself
        var zero uint64 = 0
        for i := 0; i < PBOX_ENTRIES; i += 2 {
            zero = c.decryptBlock(zero)
            c.pbox[i] = uint32(zero >> 32)
            c.pbox[i+1] = uint32(zero)
        }
        
        // Encrypt S-box 1
        for i := 0; i < SBOX_ENTRIES; i += 2 {
            zero = c.decryptBlock(zero)
            c.sbox1[i] = uint32(zero >> 32)
            c.sbox1[i+1] = uint32(zero)
        }
        
        // Encrypt S-box 2
        for i := 0; i < SBOX_ENTRIES; i += 2 {
            zero = c.decryptBlock(zero)
            c.sbox2[i] = uint32(zero >> 32)
            c.sbox2[i+1] = uint32(zero)
        }
        
        // Encrypt S-box 3
        for i := 0; i < SBOX_ENTRIES; i += 2 {
            zero = c.decryptBlock(zero)
            c.sbox3[i] = uint32(zero >> 32)
            c.sbox3[i+1] = uint32(zero)
        }
        
        // Encrypt S-box 4
        for i := 0; i < SBOX_ENTRIES; i += 2 {
            zero = c.decryptBlock(zero)
            c.sbox4[i] = uint32(zero >> 32)
            c.sbox4[i+1] = uint32(zero)
        }
    }
    
    c.cbcIV = 0
    return c
}

// NewMUSBlowfishCipherPreCalc creates a cipher using pre-calculated boxes
func NewMUSBlowfishCipherPreCalc() *MUSBlowfishCipher {
    c := &MUSBlowfishCipher{}
    c.Reset()
    return c
}

// Reset resets the cipher to use pre-calculated boxes
func (c *MUSBlowfishCipher) Reset() {
    copy(c.pbox[:], precalcPbox[:])
    copy(c.sbox1[:], precalcSbox1[:])
    copy(c.sbox2[:], precalcSbox2[:])
    copy(c.sbox3[:], precalcSbox3[:])
    copy(c.sbox4[:], precalcSbox4[:])
    c.cbcIV = 0
}

// SetCBCIV sets the CBC initialization vector
func (c *MUSBlowfishCipher) SetCBCIV(iv uint64) {
    c.cbcIV = iv
}

// cipherBlockCBC is the core of the SMUS bug - uses decryptBlock for both encrypt and decrypt
func (c *MUSBlowfishCipher) cipherBlockCBC(block uint64) uint64 {
    c.cbcIV = c.decryptBlock(c.cbcIV)
    return block ^ c.cbcIV
}

// Encrypt encrypts data in place
func (c *MUSBlowfishCipher) Encrypt(data []byte) {
    length := len(data)
    
    // Process full 8-byte blocks
    for i := 0; i < length-7; i += 8 {
        block := binary.BigEndian.Uint64(data[i : i+8])
        block = c.cipherBlockCBC(block)
        binary.BigEndian.PutUint64(data[i:i+8], block)
    }
    
    // Handle remaining bytes if any
    remaining := length & 7
    if remaining > 0 {
        // Create padded block
        var lastBlock uint64 = 0
        offset := length - remaining
        
        // Copy remaining bytes
        for j := 0; j < remaining; j++ {
            lastBlock = (lastBlock << 8) | uint64(data[offset+j])
        }
        
        // Pad with spaces (0x20)
        for j := 0; j < 8-remaining; j++ {
            lastBlock = (lastBlock << 8) | 0x20
        }
        
        // Encrypt the padded block
        lastBlock = c.cipherBlockCBC(lastBlock)
        
        // Copy back only the needed bytes
        for j := 0; j < remaining; j++ {
            data[offset+j] = byte(lastBlock >> uint(56-j*8))
        }
    }
}

// Decrypt decrypts data in place (uses same function as encrypt due to XOR properties)
func (c *MUSBlowfishCipher) Decrypt(data []byte) {
    // Due to the SMUS bug, decrypt is identical to encrypt!
    c.Encrypt(data)
}

// decryptBlock performs the Blowfish decryption round
func (c *MUSBlowfishCipher) decryptBlock(block uint64) uint64 {
    hi := uint32(block >> 32)
    lo := uint32(block)
    
    hi ^= c.pbox[17]
    lo ^= (((c.sbox1[hi>>24] + c.sbox2[(hi>>16)&0xff]) ^ c.sbox3[(hi>>8)&0xff]) + c.sbox4[hi&0xff]) ^ c.pbox[16]
    hi ^= (((c.sbox1[lo>>24] + c.sbox2[(lo>>16)&0xff]) ^ c.sbox3[(lo>>8)&0xff]) + c.sbox4[lo&0xff]) ^ c.pbox[15]
    lo ^= (((c.sbox1[hi>>24] + c.sbox2[(hi>>16)&0xff]) ^ c.sbox3[(hi>>8)&0xff]) + c.sbox4[hi&0xff]) ^ c.pbox[14]
    hi ^= (((c.sbox1[lo>>24] + c.sbox2[(lo>>16)&0xff]) ^ c.sbox3[(lo>>8)&0xff]) + c.sbox4[lo&0xff]) ^ c.pbox[13]
    lo ^= (((c.sbox1[hi>>24] + c.sbox2[(hi>>16)&0xff]) ^ c.sbox3[(hi>>8)&0xff]) + c.sbox4[hi&0xff]) ^ c.pbox[12]
    hi ^= (((c.sbox1[lo>>24] + c.sbox2[(lo>>16)&0xff]) ^ c.sbox3[(lo>>8)&0xff]) + c.sbox4[lo&0xff]) ^ c.pbox[11]
    lo ^= (((c.sbox1[hi>>24] + c.sbox2[(hi>>16)&0xff]) ^ c.sbox3[(hi>>8)&0xff]) + c.sbox4[hi&0xff]) ^ c.pbox[10]
    hi ^= (((c.sbox1[lo>>24] + c.sbox2[(lo>>16)&0xff]) ^ c.sbox3[(lo>>8)&0xff]) + c.sbox4[lo&0xff]) ^ c.pbox[9]
    lo ^= (((c.sbox1[hi>>24] + c.sbox2[(hi>>16)&0xff]) ^ c.sbox3[(hi>>8)&0xff]) + c.sbox4[hi&0xff]) ^ c.pbox[8]
    hi ^= (((c.sbox1[lo>>24] + c.sbox2[(lo>>16)&0xff]) ^ c.sbox3[(lo>>8)&0xff]) + c.sbox4[lo&0xff]) ^ c.pbox[7]
    lo ^= (((c.sbox1[hi>>24] + c.sbox2[(hi>>16)&0xff]) ^ c.sbox3[(hi>>8)&0xff]) + c.sbox4[hi&0xff]) ^ c.pbox[6]
    hi ^= (((c.sbox1[lo>>24] + c.sbox2[(lo>>16)&0xff]) ^ c.sbox3[(lo>>8)&0xff]) + c.sbox4[lo&0xff]) ^ c.pbox[5]
    lo ^= (((c.sbox1[hi>>24] + c.sbox2[(hi>>16)&0xff]) ^ c.sbox3[(hi>>8)&0xff]) + c.sbox4[hi&0xff]) ^ c.pbox[4]
    hi ^= (((c.sbox1[lo>>24] + c.sbox2[(lo>>16)&0xff]) ^ c.sbox3[(lo>>8)&0xff]) + c.sbox4[lo&0xff]) ^ c.pbox[3]
    lo ^= (((c.sbox1[hi>>24] + c.sbox2[(hi>>16)&0xff]) ^ c.sbox3[(hi>>8)&0xff]) + c.sbox4[hi&0xff]) ^ c.pbox[2]
    hi ^= (((c.sbox1[lo>>24] + c.sbox2[(lo>>16)&0xff]) ^ c.sbox3[(lo>>8)&0xff]) + c.sbox4[lo&0xff])
    
    // Final swap
    lo, hi = hi^c.pbox[1], lo^c.pbox[0]
    
    return (uint64(lo) << 32) | uint64(hi)
}

// InitGlobalBoxes initializes the pre-calculated boxes with a key
func InitGlobalBoxes(key string) {
    temp := NewMUSBlowfishCipher([]byte(key))
    copy(precalcPbox[:], temp.pbox[:])
    copy(precalcSbox1[:], temp.sbox1[:])
    copy(precalcSbox2[:], temp.sbox2[:])
    copy(precalcSbox3[:], temp.sbox3[:])
    copy(precalcSbox4[:], temp.sbox4[:])
}