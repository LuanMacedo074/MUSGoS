package ports

type Cipher interface {
	Encrypt(data []byte) []byte
	Decrypt(data []byte) []byte
}
