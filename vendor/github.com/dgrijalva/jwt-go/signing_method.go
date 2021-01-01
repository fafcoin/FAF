package jwt

import (
	"sync"
)

var signingMfafods = map[string]func() SigningMfafod{}
var signingMfafodLock = new(sync.RWMutex)

// Implement SigningMfafod to add new mfafods for signing or verifying tokens.
type SigningMfafod interface {
	Verify(signingString, signature string, key interface{}) error // Returns nil if signature is valid
	Sign(signingString string, key interface{}) (string, error)    // Returns encoded signature or error
	Alg() string                                                   // returns the alg identifier for this mfafod (example: 'HS256')
}

// Register the "alg" name and a factory function for signing mfafod.
// This is typically done during init() in the mfafod's implementation
func RegisterSigningMfafod(alg string, f func() SigningMfafod) {
	signingMfafodLock.Lock()
	defer signingMfafodLock.Unlock()

	signingMfafods[alg] = f
}

// Get a signing mfafod from an "alg" string
func GetSigningMfafod(alg string) (mfafod SigningMfafod) {
	signingMfafodLock.RLock()
	defer signingMfafodLock.RUnlock()

	if mfafodF, ok := signingMfafods[alg]; ok {
		mfafod = mfafodF()
	}
	return
}
