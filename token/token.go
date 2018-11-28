package token

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
)

var (
	// Error indicating that a token was malformed.
	ErrInvalidToken = errors.New("Invalid token.")

	// Indicating that verification failed, because the token was
	// incorrect.
	ErrIncorrectToken = errors.New("Incorrect token.")
)

// A cryptographically random 128-bit value.
type Token [128 / 8]byte

// Generate a new, cryptographically random token.
func New() (Token, error) {
	var ret Token
	_, err := rand.Read(ret[:])
	return ret, err
}

func (t Token) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%0x", t)), nil
}

func (t *Token) UnmarshalText(text []byte) error {
	if len(text) != 2*len(t[:]) {
		// wrong number of characters.
		return ErrInvalidToken
	}
	for _, char := range text {
		if !isHexDigit(char) {
			return ErrInvalidToken
		}
	}
	var buf []byte
	_, err := fmt.Fscanf(bytes.NewBuffer(text), "%32x", &buf)
	if err != nil {
		return ErrInvalidToken
	}
	copy(t[:], buf)
	return nil
}

// Return an error if the tokens do not match, nil otherwise. This is
// constant time, and thus resistant to timing sidechannels -- DO NOT
// compare the tokens for equality with (==).
func (t Token) Verify(otherTok Token) error {
	if subtle.ConstantTimeCompare(t[:], otherTok[:]) == 1 {
		return nil
	}
	return ErrIncorrectToken
}

func isHexDigit(char byte) bool {
	return char >= '0' && char <= '9' ||
		char >= 'a' && char <= 'f' ||
		char >= 'A' && char <= 'F'
}
