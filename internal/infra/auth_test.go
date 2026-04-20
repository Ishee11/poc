package infra

import "testing"

func TestArgon2IDPasswordHasher(t *testing.T) {
	hasher := Argon2IDPasswordHasher{}

	hash, err := hasher.HashPassword("long-secure-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hash == "long-secure-password" {
		t.Fatal("password hash must not equal raw password")
	}
	if !hasher.VerifyPassword("long-secure-password", hash) {
		t.Fatal("expected password verification to pass")
	}
	if hasher.VerifyPassword("wrong-password", hash) {
		t.Fatal("expected wrong password verification to fail")
	}
}

func TestArgon2IDPasswordHasherRejectsInvalidHash(t *testing.T) {
	hasher := Argon2IDPasswordHasher{}

	if hasher.VerifyPassword("password", "not-a-valid-hash") {
		t.Fatal("invalid hash must not verify")
	}
}
