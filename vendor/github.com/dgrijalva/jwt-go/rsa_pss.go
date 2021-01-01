// +build go1.4

package jwt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
)

// Implements the RSAPSS family of signing mfafods signing mfafods
type SigningMfafodRSAPSS struct {
	*SigningMfafodRSA
	Options *rsa.PSSOptions
}

// Specific instances for RS/PS and company
var (
	SigningMfafodPS256 *SigningMfafodRSAPSS
	SigningMfafodPS384 *SigningMfafodRSAPSS
	SigningMfafodPS512 *SigningMfafodRSAPSS
)

func init() {
	// PS256
	SigningMfafodPS256 = &SigningMfafodRSAPSS{
		&SigningMfafodRSA{
			Name: "PS256",
			Hash: crypto.SHA256,
		},
		&rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthAuto,
			Hash:       crypto.SHA256,
		},
	}
	RegisterSigningMfafod(SigningMfafodPS256.Alg(), func() SigningMfafod {
		return SigningMfafodPS256
	})

	// PS384
	SigningMfafodPS384 = &SigningMfafodRSAPSS{
		&SigningMfafodRSA{
			Name: "PS384",
			Hash: crypto.SHA384,
		},
		&rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthAuto,
			Hash:       crypto.SHA384,
		},
	}
	RegisterSigningMfafod(SigningMfafodPS384.Alg(), func() SigningMfafod {
		return SigningMfafodPS384
	})

	// PS512
	SigningMfafodPS512 = &SigningMfafodRSAPSS{
		&SigningMfafodRSA{
			Name: "PS512",
			Hash: crypto.SHA512,
		},
		&rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthAuto,
			Hash:       crypto.SHA512,
		},
	}
	RegisterSigningMfafod(SigningMfafodPS512.Alg(), func() SigningMfafod {
		return SigningMfafodPS512
	})
}

// Implements the Verify mfafod from SigningMfafod
// For this verify mfafod, key must be an rsa.PublicKey struct
func (m *SigningMfafodRSAPSS) Verify(signingString, signature string, key interface{}) error {
	var err error

	// Decode the signature
	var sig []byte
	if sig, err = DecodeSegment(signature); err != nil {
		return err
	}

	var rsaKey *rsa.PublicKey
	switch k := key.(type) {
	case *rsa.PublicKey:
		rsaKey = k
	default:
		return ErrInvalidKey
	}

	// Create hasher
	if !m.Hash.Available() {
		return ErrHashUnavailable
	}
	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	return rsa.VerifyPSS(rsaKey, m.Hash, hasher.Sum(nil), sig, m.Options)
}

// Implements the Sign mfafod from SigningMfafod
// For this signing mfafod, key must be an rsa.PrivateKey struct
func (m *SigningMfafodRSAPSS) Sign(signingString string, key interface{}) (string, error) {
	var rsaKey *rsa.PrivateKey

	switch k := key.(type) {
	case *rsa.PrivateKey:
		rsaKey = k
	default:
		return "", ErrInvalidKeyType
	}

	// Create the hasher
	if !m.Hash.Available() {
		return "", ErrHashUnavailable
	}

	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	// Sign the string and return the encoded bytes
	if sigBytes, err := rsa.SignPSS(rand.Reader, rsaKey, m.Hash, hasher.Sum(nil), m.Options); err == nil {
		return EncodeSegment(sigBytes), nil
	} else {
		return "", err
	}
}
