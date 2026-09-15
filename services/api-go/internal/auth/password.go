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

// Argon2id parameters — identical to @node-rs/argon2 defaults, so hashes are
// interchangeable in both directions. Verify reads the params from the PHC
// string itself; these defaults apply only to newly created hashes.
const (
	hashTime    uint32 = 2
	hashMemory  uint32 = 19456 // KiB
	hashThreads uint8  = 1
	hashKeyLen  uint32 = 32
	hashSaltLen       = 16
)

var b64 = base64.RawStdEncoding

// HashPassword hashes plaintext with argon2id, returning a PHC string in the
// exact shape node-rs produces: $argon2id$v=19$m=..,t=..,p=..$salt$hash.
func HashPassword(plain string) (string, error) {
	salt := make([]byte, hashSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(plain), salt, hashTime, hashMemory, hashThreads, hashKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		hashMemory, hashTime, hashThreads,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword reports whether plain matches the PHC hash. Malformed input
// returns false (never an error) — mirroring password.ts catch→false.
func VerifyPassword(hashed, plain string) bool {
	params, salt, key, err := parsePHC(hashed)
	if err != nil {
		return false
	}
	computed := argon2.IDKey([]byte(plain), salt, params.time, params.memory, params.threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(computed, key) == 1
}

type argonParams struct {
	time    uint32
	memory  uint32
	threads uint8
}

func parsePHC(phc string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return argonParams{}, nil, nil, errors.New("auth: unsupported PHC format")
	}
	var p argonParams
	for _, kv := range strings.Split(parts[3], ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return argonParams{}, nil, nil, errors.New("auth: bad PHC params")
		}
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return argonParams{}, nil, nil, errors.New("auth: bad PHC param value")
		}
		switch k {
		case "m":
			p.memory = uint32(n)
		case "t":
			p.time = uint32(n)
		case "p":
			if n < 1 || n > 255 {
				return argonParams{}, nil, nil, errors.New("auth: bad PHC parallelism")
			}
			p.threads = uint8(n)
		default:
			return argonParams{}, nil, nil, errors.New("auth: unknown PHC param")
		}
	}
	if p.time == 0 || p.memory == 0 || p.threads == 0 {
		return argonParams{}, nil, nil, errors.New("auth: zero PHC param")
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, errors.New("auth: bad PHC salt")
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, errors.New("auth: bad PHC hash")
	}
	return p, salt, key, nil
}
