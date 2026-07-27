package internal

import (
	"crypto/rand"
	"crypto/sha1"
	"math/big"
	"strings"
)

var srpG = big.NewInt(7)

func srpN() *big.Int {
	N := new(big.Int)
	N.SetString("894B645E89E1535BBDAD5B8B290650530801B18EBFBF5E8FAB3C82872A3E9BB7", 16)
	return N
}

// generateVerifier generates a fresh random salt and computes the matching
// SRP-6 verifier - used when creating an account.
func generateVerifier(username, password string) (salt, verifier []byte, err error) {
	salt = make([]byte, 32)
	if _, err = rand.Read(salt); err != nil {
		return nil, nil, err
	}
	verifier, err = computeVerifier(username, password, salt)
	if err != nil {
		return nil, nil, err
	}
	return salt, verifier, nil
}

// computeVerifier computes the SRP-6 verifier for username/password against
// an existing salt - used to re-derive a verifier for comparison against a
// stored one (login / old-password checks), and as the implementation
// backing generateVerifier.
func computeVerifier(username, password string, salt []byte) (verifier []byte, err error) {
	h := sha1.New()
	h.Write([]byte(strings.ToUpper(username) + ":" + strings.ToUpper(password)))
	innerHash := h.Sum(nil)

	h = sha1.New()
	h.Write(salt)
	h.Write(innerHash)
	xHash := h.Sum(nil)

	// Reverse bytes: treat the SHA1 output as a little-endian integer
	for lo, hi := 0, len(xHash)-1; lo < hi; lo, hi = lo+1, hi-1 {
		xHash[lo], xHash[hi] = xHash[hi], xHash[lo]
	}
	x := new(big.Int).SetBytes(xHash)

	v := new(big.Int).Exp(srpG, x, srpN())

	// Zero-pad to 32 bytes (big-endian), then reverse to little-endian for storage
	vBE := make([]byte, 32)
	copy(vBE[32-len(v.Bytes()):], v.Bytes())
	verifier = make([]byte, 32)
	for idx, b := range vBE {
		verifier[31-idx] = b
	}

	return verifier, nil
}
