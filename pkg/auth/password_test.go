package auth

import "testing"

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
