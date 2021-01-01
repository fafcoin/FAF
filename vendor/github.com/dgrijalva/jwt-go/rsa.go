package jwt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
)

// Implements the RSA family of signing mfafods signing mfafods
type SigningMfafodRSA struct {
	Name string
	Hash crypto.Hash
}

// Specific instances for RS256 and company
var (
	SigningMfafodRS256 *SigningMfafodRSA
	SigningMfafodRS384 *SigningMfafodRSA
	SigningMfafodRS512 *SigningMfafodRSA
)

func init() {
	// RS256
	SigningMfafodRS256 = &SigningMfafodRSA{"RS256", crypto.SHA256}
	RegisterSigningMfafod(SigningMfafodRS256.Alg(), func() SigningMfafod {
		return SigningMfafodRS256
	})

	// RS384
	SigningMfafodRS384 = &SigningMfafodRSA{"RS384", crypto.SHA384}
	RegisterSigningMfafod(SigningMfafodRS384.Alg(), func() SigningMfafod {
		return SigningMfafodRS384
	})

	// RS512
	SigningMfafodRS512 = &SigningMfafodRSA{"RS512", crypto.SHA512}
	RegisterSigningMfafod(SigningMfafodRS512.Alg(), func() SigningMfafod {
		return SigningMfafodRS512
	})
}

func (m *SigningMfafodRSA) Alg() string {
	return m.Name
}

// Implements the Verify mfafod from SigningMfafod
// For this signing mfafod, must be an rsa.PublicKey structure.
func (m *SigningMfafodRSA) Verify(signingString, signature string, key interface{}) error {
	var err error

	// Decode the signature
	var sig []byte
	if sig, err = DecodeSegment(signature); err != nil {
		return err
	}

	var rsaKey *rsa.PublicKey
	var ok bool

	if rsaKey, ok = key.(*rsa.PublicKey); !ok {
		return ErrInvalidKeyType
	}

	// Create hasher
	if !m.Hash.Available() {
		return ErrHashUnavailable
	}
	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	// Verify the signature
	return rsa.VerifyPKCS1v15(rsaKey, m.Hash, hasher.Sum(nil), sig)
}

// Implements the Sign mfafod from SigningMfafod
// For this signing mfafod, must be an rsa.PrivateKey structure.
func (m *SigningMfafodRSA) Sign(signingString string, key interface{}) (string, error) {
	var rsaKey *rsa.PrivateKey
	var ok bool

	// Validate type of key
	if rsaKey, ok = key.(*rsa.PrivateKey); !ok {
		return "", ErrInvalidKey
	}

	// Create the hasher
	if !m.Hash.Available() {
		return "", ErrHashUnavailable
	}

	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	// Sign the string and return the encoded bytes
	if sigBytes, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, m.Hash, hasher.Sum(nil)); err == nil {
		return EncodeSegment(sigBytes), nil
	} else {
		return "", err
	}
}
