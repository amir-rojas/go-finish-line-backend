package security_test

import (
	"strings"
	"testing"

	"finish-line/internal/common/security"
	"finish-line/internal/user/ports"
)

// Compile-time proof that BcryptHasher satisfies the user module's port.
var _ ports.PasswordHasher = (*security.BcryptHasher)(nil)

func TestBcryptHasher(t *testing.T) {
	hasher := security.NewBcryptHasher()

	hash, err := hasher.Hash("secure-password")
	if err != nil {
		t.Fatalf("Hash() unexpected error: %v", err)
	}
	if hash == "secure-password" {
		t.Fatal("Hash() returned the plain password")
	}

	t.Run("correct password verifies", func(t *testing.T) {
		ok, err := hasher.Verify("secure-password", hash)
		if err != nil {
			t.Fatalf("Verify() unexpected error: %v", err)
		}
		if !ok {
			t.Error("Verify() = false for the correct password")
		}
	})

	t.Run("wrong password does not verify", func(t *testing.T) {
		ok, err := hasher.Verify("wrong-password", hash)
		if err != nil {
			t.Fatalf("Verify() unexpected error: %v", err)
		}
		if ok {
			t.Error("Verify() = true for a wrong password")
		}
	})

	t.Run("malformed hash returns error", func(t *testing.T) {
		_, err := hasher.Verify("whatever", "not-a-bcrypt-hash")
		if err == nil {
			t.Error("Verify() expected error for malformed hash")
		}
	})

	t.Run("over 72 bytes is rejected by Hash", func(t *testing.T) {
		_, err := hasher.Hash(strings.Repeat("a", 73))
		if err == nil {
			t.Error("Hash() expected error for input over bcrypt's 72-byte limit")
		}
	})
}
