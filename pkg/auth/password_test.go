package auth

import (
	"errors"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("password", PasswordConfig{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("password", hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("password should verify")
	}
	ok, err = VerifyPassword("wrong", hash)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("wrong password should not verify")
	}
}

func TestHashPasswordUsesDefaultsWhenMemoryIsZero(t *testing.T) {
	hash, err := HashPassword("password", PasswordConfig{Memory: 0})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("password", hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("password should verify with default config")
	}
}

func TestVerifyPasswordRejectsInvalidHashes(t *testing.T) {
	cases := []string{
		"",
		"not-a-hash",
		"$argon2id$v=18$m=1,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=1,t=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=x,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=1,t=1,p=1$not base64$aGFzaA",
		"$argon2id$v=19$m=1,t=1,p=1$c2FsdA$not base64",
	}
	for _, encoded := range cases {
		ok, err := VerifyPassword("password", encoded)
		if ok {
			t.Fatalf("VerifyPassword(%q) unexpectedly succeeded", encoded)
		}
		if !errors.Is(err, ErrInvalidHash) {
			t.Fatalf("VerifyPassword(%q) err = %v, want ErrInvalidHash", encoded, err)
		}
	}
}
