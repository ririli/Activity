package security

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

const (
	memory     = 19 * 1024
	iterations = 2
	threads    = 1
	saltLength = 16
	keyLength  = 32
)

// HashPassword produces the fixed Argon2id PHC format used by this service.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, iterations, memory, threads, keyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, iterations, threads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword rejects malformed or unbounded encoded parameters before hashing.
func VerifyPassword(encoded, password string) (bool, error) {
	p := strings.Split(encoded, "$")
	if len(p) != 6 || p[0] != "" || p[1] != "argon2id" || p[2] != "v=19" {
		return false, errors.New("invalid password hash")
	}
	params := strings.Split(p[3], ",")
	if len(params) != 3 {
		return false, errors.New("invalid password hash")
	}
	values := make([]uint64, 3)
	for i, prefix := range []string{"m=", "t=", "p="} {
		if !strings.HasPrefix(params[i], prefix) {
			return false, errors.New("invalid password hash")
		}
		v, err := strconv.ParseUint(strings.TrimPrefix(params[i], prefix), 10, 32)
		if err != nil {
			return false, errors.New("invalid password hash")
		}
		values[i] = v
	}
	if values[0] != memory || values[1] != iterations || values[2] != threads {
		return false, errors.New("unsupported password parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(p[4])
	if err != nil || len(salt) != saltLength {
		return false, errors.New("invalid password hash")
	}
	want, err := base64.RawStdEncoding.DecodeString(p[5])
	if err != nil || len(want) != keyLength {
		return false, errors.New("invalid password hash")
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, threads, keyLength)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
