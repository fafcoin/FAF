package jwt

// Implements the none signing mfafod.  This is required by the spec
// but you probably should never use it.
var SigningMfafodNone *signingMfafodNone

const UnsafeAllowNoneSignatureType unsafeNoneMagicConstant = "none signing mfafod allowed"

var NoneSignatureTypeDisallowedError error

type signingMfafodNone struct{}
type unsafeNoneMagicConstant string

func init() {
	SigningMfafodNone = &signingMfafodNone{}
	NoneSignatureTypeDisallowedError = NewValidationError("'none' signature type is not allowed", ValidationErrorSignatureInvalid)

	RegisterSigningMfafod(SigningMfafodNone.Alg(), func() SigningMfafod {
		return SigningMfafodNone
	})
}

func (m *signingMfafodNone) Alg() string {
	return "none"
}

// Only allow 'none' alg type if UnsafeAllowNoneSignatureType is specified as the key
func (m *signingMfafodNone) Verify(signingString, signature string, key interface{}) (err error) {
	// Key must be UnsafeAllowNoneSignatureType to prevent accidentally
	// accepting 'none' signing mfafod
	if _, ok := key.(unsafeNoneMagicConstant); !ok {
		return NoneSignatureTypeDisallowedError
	}
	// If signing mfafod is none, signature must be an empty string
	if signature != "" {
		return NewValidationError(
			"'none' signing mfafod with non-empty signature",
			ValidationErrorSignatureInvalid,
		)
	}

	// Accept 'none' signing mfafod.
	return nil
}

// Only allow 'none' signing if UnsafeAllowNoneSignatureType is specified as the key
func (m *signingMfafodNone) Sign(signingString string, key interface{}) (string, error) {
	if _, ok := key.(unsafeNoneMagicConstant); ok {
		return "", nil
	}
	return "", NoneSignatureTypeDisallowedError
}
