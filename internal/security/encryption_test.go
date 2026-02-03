package security

import (
	"testing"
)

func TestHashSHA256(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple string",
			input:    "hello",
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "Phone number",
			input:    "+989123456789",
			expected: "c77b80f145ea25a19e1ca97f5fc1f2ea3b205de55b24904e47331da8151e9d9c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashSHA256(tt.input)
			if result != tt.expected {
				t.Errorf("HashSHA256(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEncryptDecryptAES256(t *testing.T) {
	key := []byte("12345678901234567890123456789012") // 32 bytes

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "Simple text",
			plaintext: "Hello, World!",
		},
		{
			name:      "Persian text",
			plaintext: "سلام دنیا",
		},
		{
			name:      "Empty string",
			plaintext: "",
		},
		{
			name:      "Long text",
			plaintext: "This is a very long text that should be encrypted and decrypted successfully without any issues.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := EncryptAES256(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptAES256() error = %v", err)
			}

			// Decrypt
			decrypted, err := DecryptAES256(encrypted, key)
			if err != nil {
				t.Fatalf("DecryptAES256() error = %v", err)
			}

			// Verify
			if decrypted != tt.plaintext {
				t.Errorf("Decrypted text = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptAES256_InvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{
			name: "Too short",
			key:  []byte("short"),
		},
		{
			name: "Too long",
			key:  []byte("123456789012345678901234567890123"),
		},
		{
			name: "Empty",
			key:  []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptAES256("test", tt.key)
			if err == nil {
				t.Error("EncryptAES256() expected error for invalid key, got nil")
			}
		})
	}
}

func TestDecryptAES256_InvalidKey(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")
	encrypted, _ := EncryptAES256("test", validKey)

	invalidKey := []byte("00000000000000000000000000000000")
	_, err := DecryptAES256(encrypted, invalidKey)
	if err == nil {
		t.Error("DecryptAES256() expected error for wrong key, got nil")
	}
}

func TestDecryptAES256_InvalidCiphertext(t *testing.T) {
	key := []byte("12345678901234567890123456789012")

	tests := []struct {
		name       string
		ciphertext string
	}{
		{
			name:       "Invalid base64",
			ciphertext: "not-valid-base64!@#",
		},
		{
			name:       "Too short",
			ciphertext: "YWJj", // "abc" in base64
		},
		{
			name:       "Empty",
			ciphertext: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptAES256(tt.ciphertext, key)
			if err == nil {
				t.Error("DecryptAES256() expected error for invalid ciphertext, got nil")
			}
		})
	}
}
