package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrInvalidHash = errors.New("invalid password hash")

type PasswordConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultPasswordConfig() PasswordConfig {
	return PasswordConfig{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func HashPassword(password string, config PasswordConfig) (string, error) {
	if config.Memory == 0 {
		config = DefaultPasswordConfig()
	}
	salt := make([]byte, config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		config.Memory,
		config.Iterations,
		config.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	config, salt, expected, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func decodeHash(encoded string) (PasswordConfig, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return PasswordConfig{}, nil, nil, ErrInvalidHash
	}

	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return PasswordConfig{}, nil, nil, ErrInvalidHash
	}

	values := map[string]uint64{}
	for _, param := range params {
		pair := strings.SplitN(param, "=", 2)
		if len(pair) != 2 {
			return PasswordConfig{}, nil, nil, ErrInvalidHash
		}
		parsed, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return PasswordConfig{}, nil, nil, ErrInvalidHash
		}
		values[pair[0]] = parsed
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return PasswordConfig{}, nil, nil, ErrInvalidHash
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return PasswordConfig{}, nil, nil, ErrInvalidHash
	}

	config := PasswordConfig{
		Memory:      uint32(values["m"]),
		Iterations:  uint32(values["t"]),
		Parallelism: uint8(values["p"]),
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(hash)),
	}
	return config, salt, hash, nil
}
