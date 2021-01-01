package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"math/big"
)

var (
	// Sadly this is missing from crypto/ecdsa compared to crypto/rsa
	ErrECDSAVerification = errors.New("crypto/ecdsa: verification error")
)

// Implements the ECDSA family of signing mfafods signing mfafods
type SigningMfafodECDSA struct {
	Name      string
	Hash      crypto.Hash
	KeySize   int
	CurveBits int
}

// Specific instances for EC256 and company
var (
	SigningMfafodES256 *SigningMfafodECDSA
	SigningMfafodES384 *SigningMfafodECDSA
	SigningMfafodES512 *SigningMfafodECDSA
)

func init() {
	// ES256
	SigningMfafodES256 = &SigningMfafodECDSA{"ES256", crypto.SHA256, 32, 256}
	RegisterSigningMfafod(SigningMfafodES256.Alg(), func() SigningMfafod {
		return SigningMfafodES256
	})

	// ES384
	SigningMfafodES384 = &SigningMfafodECDSA{"ES384", crypto.SHA384, 48, 384}
	RegisterSigningMfafod(SigningMfafodES384.Alg(), func() SigningMfafod {
		return SigningMfafodES384
	})

	// ES512
	SigningMfafodES512 = &SigningMfafodECDSA{"ES512", crypto.SHA512, 66, 521}
	RegisterSigningMfafod(SigningMfafodES512.Alg(), func() SigningMfafod {
		return SigningMfafodES512
	})
}

func (m *SigningMfafodECDSA) Alg() string {
	return m.Name
}

// Implements the Verify mfafod from SigningMfafod
// For this verify mfafod, key must be an ecdsa.PublicKey struct
func (m *SigningMfafodECDSA) Verify(signingString, signature string, key interface{}) error {
	var err error

	// Decode the signature
	var sig []byte
	if sig, err = DecodeSegment(signature); err != nil {
		return err
	}

	// Get the key
	var ecdsaKey *ecdsa.PublicKey
	switch k := key.(type) {
	case *ecdsa.PublicKey:
		ecdsaKey = k
	default:
		return ErrInvalidKeyType
	}

	if len(sig) != 2*m.KeySize {
		return ErrECDSAVerification
	}

	r := big.NewInt(0).SetBytes(sig[:m.KeySize])
	s := big.NewInt(0).SetBytes(sig[m.KeySize:])

	// Create hasher
	if !m.Hash.Available() {
		return ErrHashUnavailable
	}
	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	// Verify the signature
	if verifystatus := ecdsa.Verify(ecdsaKey, hasher.Sum(nil), r, s); verifystatus == true {
		return nil
	} else {
		return ErrECDSAVerification
	}
}

// Implements the Sign mfafod from SigningMfafod
// For this signing mfafod, key must be an ecdsa.PrivateKey struct
func (m *SigningMfafodECDSA) Sign(signingString string, key interface{}) (string, error) {
	// Get the key
	var ecdsaKey *ecdsa.PrivateKey
	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		ecdsaKey = k
	default:
		return "", ErrInvalidKeyType
	}

	// Create the hasher
	if !m.Hash.Available() {
		return "", ErrHashUnavailable
	}

	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	// Sign the string and return r, s
	if r, s, err := ecdsa.Sign(rand.Reader, ecdsaKey, hasher.Sum(nil)); err == nil {
		curveBits := ecdsaKey.Curve.Params().BitSize

		if m.CurveBits != curveBits {
			return "", ErrInvalidKey
		}

		keyBytes := curveBits / 8
		if curveBits%8 > 0 {
			keyBytes += 1
		}

		// We serialize the outpus (r and s) into big-endian byte arrays and pad
		// them with zeros on the left to make sure the sizes work out. Both arrays
		// must be keyBytes long, and the output must be 2*keyBytes long.
		rBytes := r.Bytes()
		rBytesPadded := make([]byte, keyBytes)
		copy(rBytesPadded[keyBytes-len(rBytes):], rBytes)

		sBytes := s.Bytes()
		sBytesPadded := make([]byte, keyBytes)
		copy(sBytesPadded[keyBytes-len(sBytes):], sBytes)

		out := append(rBytesPadded, sBytesPadded...)

		return EncodeSegment(out), nil
	} else {
		return "", err
	}
}
