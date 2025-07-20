package crypto

import (
    "bytes"
    "encoding/binary"
    "encoding/hex"
    "testing"
)

func TestMUSBlowfish(t *testing.T) {
    // Initialize with the same key as the server
    key := "1289372893742894792893472938742"
    InitGlobalBoxes(key)
    
    // Test data - a simple LPropList with type 0x000A (10)
    testData := []byte{
        0x00, 0x0A, // Type: PropList (10)
        0x00, 0x00, 0x00, 0x03, // Count: 3 items
        // Add more bytes as needed
    }
    
    t.Logf("Original data: %s", hex.EncodeToString(testData))
    
    // Create cipher and encrypt
    cipher1 := NewMUSBlowfishDefault()
    encrypted := make([]byte, len(testData))
    copy(encrypted, testData)
    cipher1.Encode(encrypted)
    
    t.Logf("Encrypted data: %s", hex.EncodeToString(encrypted))
    
    // Create new cipher and decrypt  
    cipher2 := NewMUSBlowfishDefault()
    decrypted := make([]byte, len(encrypted))
    copy(decrypted, encrypted)
    cipher2.Decode(decrypted)
    
    t.Logf("Decrypted data: %s", hex.EncodeToString(decrypted))
    
    // Verify
    if !bytes.Equal(testData, decrypted) {
        t.Errorf("Decryption failed!\nExpected: %s\nGot:      %s", 
            hex.EncodeToString(testData), 
            hex.EncodeToString(decrypted))
    }
}

func TestLoginPropList(t *testing.T) {
    // Test with the login PropList format
    // [#userID: "teste", #password: "teste", #movieID:"faria"]
    
    key := "1289372893742894792893472938742"
    InitGlobalBoxes(key)
    
    // Build the PropList manually
    var buf bytes.Buffer
    
    // PropList header
    binary.Write(&buf, binary.BigEndian, uint16(0x000A)) // Type: PropList
    binary.Write(&buf, binary.BigEndian, uint32(3))      // Count: 3 items
    
    // Item 1: #userID (Symbol)
    binary.Write(&buf, binary.BigEndian, uint16(0x0002)) // Type: Symbol
    binary.Write(&buf, binary.BigEndian, uint32(6))      // Length: 6
    buf.WriteString("userID")
    
    // Value 1: "teste" (String)
    binary.Write(&buf, binary.BigEndian, uint16(0x0003)) // Type: String  
    binary.Write(&buf, binary.BigEndian, uint32(5))      // Length: 5
    buf.WriteString("teste")
    
    // Item 2: #password (Symbol)
    binary.Write(&buf, binary.BigEndian, uint16(0x0002)) // Type: Symbol
    binary.Write(&buf, binary.BigEndian, uint32(8))      // Length: 8
    buf.WriteString("password")
    
    // Value 2: "teste" (String)
    binary.Write(&buf, binary.BigEndian, uint16(0x0003)) // Type: String
    binary.Write(&buf, binary.BigEndian, uint32(5))      // Length: 5
    buf.WriteString("teste")
    
    // Item 3: #movieID (Symbol)
    binary.Write(&buf, binary.BigEndian, uint16(0x0002)) // Type: Symbol
    binary.Write(&buf, binary.BigEndian, uint32(7))      // Length: 7
    buf.WriteString("movieID")
    
    // Value 3: "faria" (String)
    binary.Write(&buf, binary.BigEndian, uint16(0x0003)) // Type: String
    binary.Write(&buf, binary.BigEndian, uint32(5))      // Length: 5
    buf.WriteString("faria")
    
    testData := buf.Bytes()
    t.Logf("Original PropList (%d bytes): %s", len(testData), hex.EncodeToString(testData))
    
    // Encrypt
    cipher1 := NewMUSBlowfishDefault()
    encrypted := make([]byte, len(testData))
    copy(encrypted, testData)
    cipher1.Encode(encrypted)
    
    t.Logf("Encrypted: %s", hex.EncodeToString(encrypted))
    
    // Decrypt
    cipher2 := NewMUSBlowfishDefault()
    decrypted := make([]byte, len(encrypted))
    copy(decrypted, encrypted)
    cipher2.Decode(decrypted)
    
    t.Logf("Decrypted: %s", hex.EncodeToString(decrypted))
    
    // Verify first bytes
    if decrypted[0] == 0x00 && decrypted[1] == 0x0A {
        t.Log("✓ Correctly decrypted to PropList type")
    } else {
        t.Errorf("✗ Wrong type after decryption: %02X %02X", decrypted[0], decrypted[1])
    }
}

func TestKnownVector(t *testing.T) {
    // Test with the actual encrypted data from the log
    key := "1289372893742894792893472938742"
    InitGlobalBoxes(key)
    
    // The encrypted data from your log
    encrypted := []byte{
        0x8C, 0xBD, 0x61, 0xCA, 0x11, 0x55, 0xA0, 0x56,
        0xF8, 0x6C, 0xD8, 0xC4, 0x40, 0x43, 0x97, 0x5F,
        // ... add more bytes
    }
    
    cipher := NewMUSBlowfishDefault()
    decrypted := make([]byte, len(encrypted))
    copy(decrypted, encrypted)
    cipher.Decode(decrypted)
    
    t.Logf("First 16 bytes decrypted: %s", hex.EncodeToString(decrypted[:16]))
    
    // Check if it starts with a valid Lingo type
    if decrypted[0] == 0x00 && decrypted[1] == 0x0A {
        t.Log("Looks like a PropList!")
    } else {
        t.Logf("Unknown type: %02X %02X", decrypted[0], decrypted[1])
    }
}