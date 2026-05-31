package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"testing"
)

func TestCFB8EncryptDecrypt(t *testing.T) {
	key := make([]byte, 16)
	rand.Read(key)

	block, _ := aes.NewCipher(key)
	encrypt := NewCFB8Encrypt(block, key)
	decrypt := NewCFB8Decrypt(block, key)

	plaintext := []byte("Hello, Minecraft! This is a test message for CFB8.")
	ciphertext := make([]byte, len(plaintext))
	decrypted := make([]byte, len(plaintext))

	encrypt.XORKeyStream(ciphertext, plaintext)
	decrypt.XORKeyStream(decrypted, ciphertext)

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("CFB8 round trip failed:\n  got:  %x\n  want: %x", decrypted, plaintext)
	}
}

func TestCFB8Empty(t *testing.T) {
	key := make([]byte, 16)
	rand.Read(key)
	block, _ := aes.NewCipher(key)

	encrypt := NewCFB8Encrypt(block, key)
	encrypt.XORKeyStream(nil, nil)
}

func TestCFB8SmallData(t *testing.T) {
	key := make([]byte, 16)
	rand.Read(key)
	block, _ := aes.NewCipher(key)

	encrypt := NewCFB8Encrypt(block, key)
	decrypt := NewCFB8Decrypt(block, key)

	plaintext := []byte("a")
	ciphertext := make([]byte, 1)
	decrypted := make([]byte, 1)

	encrypt.XORKeyStream(ciphertext, plaintext)
	decrypt.XORKeyStream(decrypted, ciphertext)

	if plaintext[0] != decrypted[0] {
		t.Errorf("single byte CFB8: got %d, want %d", decrypted[0], plaintext[0])
	}
}

func TestCFB8MultiBlock(t *testing.T) {
	key := make([]byte, 16)
	rand.Read(key)
	block, _ := aes.NewCipher(key)

	encrypt := NewCFB8Encrypt(block, key)
	decrypt := NewCFB8Decrypt(block, key)

	// More than 2 blocks to test bulk path
	plaintext := make([]byte, 160)
	rand.Read(plaintext)

	ciphertext := make([]byte, len(plaintext))
	decrypted := make([]byte, len(plaintext))

	encrypt.XORKeyStream(ciphertext, plaintext)
	decrypt.XORKeyStream(decrypted, ciphertext)

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("multi-block CFB8 round trip failed")
	}
}

func TestKeyExchange(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 16 {
		t.Errorf("key length = %d, want 16", len(key))
	}
}

func TestCreateCipher(t *testing.T) {
	key := make([]byte, 16)
	rand.Read(key)

	encrypt, decrypt, err := CreateCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	if encrypt == nil || decrypt == nil {
		t.Fatal("cipher streams should not be nil")
	}
}

func BenchmarkCFB8Small(b *testing.B) {
	key := make([]byte, 16)
	rand.Read(key)
	block, _ := aes.NewCipher(key)

	plaintext := make([]byte, 64)
	rand.Read(plaintext)
	ciphertext := make([]byte, 64)

	for i := 0; i < b.N; i++ {
		enc := NewCFB8Encrypt(block, key)
		enc.XORKeyStream(ciphertext, plaintext)
	}
}

func BenchmarkCFB8Large(b *testing.B) {
	key := make([]byte, 16)
	rand.Read(key)
	block, _ := aes.NewCipher(key)

	plaintext := make([]byte, 8192)
	rand.Read(plaintext)
	ciphertext := make([]byte, 8192)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc := NewCFB8Encrypt(block, key)
		enc.XORKeyStream(ciphertext, plaintext)
	}
}
