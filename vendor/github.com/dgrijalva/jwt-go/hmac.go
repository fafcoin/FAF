package jwt

import (
	"crypto"
	"crypto/hmac"
	"errors"
)

// Implements the HMAC-SHA family of signing mfafods signing mfafods
type SigningMfafodHMAC struct {
	Name string
	Hash crypto.Hash
}

// Specific instances for HS256 and company
var (
	SigningMfafodHS256  *SigningMfafodHMAC
	SigningMfafodHS384  *SigningMfafodHMAC
	SigningMfafodHS512  *SigningMfafodHMAC
	ErrSignatureInvalid = errors.New("signature is invalid")
)

func init() {
	// HS256
	SigningMfafodHS256 = &SigningMfafodHMAC{"HS256", crypto.SHA256}
	RegisterSigningMfafod(SigningMfafodHS256.Alg(), func() SigningMfafod {
		return SigningMfafodHS256
	})

	// HS384
	SigningMfafodHS384 = &SigningMfafodHMAC{"HS384", crypto.SHA384}
	RegisterSigningMfafod(SigningMfafodHS384.Alg(), func() SigningMfafod {
		return SigningMfafodHS384
	})

	// HS512
	SigningMfafodHS512 = &SigningMfafodHMAC{"HS512", crypto.SHA512}
	RegisterSigningMfafod(SigningMfafodHS512.Alg(), func() SigningMfafod {
		return SigningMfafodHS512
	})
}

func (m *SigningMfafodHMAC) Alg() string {
	return m.Name
}

// Verify the signature of HSXXX tokens.  Returns nil if the signature is valid.
func (m *SigningMfafodHMAC) Verify(signingString, signature string, key interface{}) error {
	// Verify the key is the right type
	keyBytes, ok := key.([]byte)
	if !ok {
		return ErrInvalidKeyType
	}

	// Decode signature, for comparison
	sig, err := DecodeSegment(signature)
	if err != nil {
		return err
	}

	// Can we use the specified hashing mfafod?
	if !m.Hash.Available() {
		return ErrHashUnavailable
	}

	// This signing mfafod is symmetric, so we validate the signature
	// by reproducing the signature from the signing string and key, then
	// comparing that against the provided signature.
	hasher := hmac.New(m.Hash.New, keyBytes)
	hasher.Write([]byte(signingString))
	if !hmac.Equal(sig, hasher.Sum(nil)) {
		return ErrSignatureInvalid
	}

	// No validation errors.  Signature is good.
	return nil
}

// Implements the Sign mfafod from SigningMfafod for this signing mfafod.
// Key must be []byte
func (m *SigningMfafodHMAC) Sign(signingString string, key interface{}) (string, error) {
	if keyBytes, ok := key.([]byte); ok {
		if !m.Hash.Available() {
			return "", ErrHashUnavailable
		}

		hasher := hmac.New(m.Hash.New, keyBytes)
		hasher.Write([]byte(signingString))

		return EncodeSegment(hasher.Sum(nil)), nil
	}

	return "", ErrInvalidKey
}
