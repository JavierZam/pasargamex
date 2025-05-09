package firebase

import (
	"context"
)

// GenerateLongLivedToken membuat token abadi untuk keperluan development
func (f *FirebaseAuthClient) GenerateLongLivedToken(ctx context.Context, uid string) (string, error) {
	// Buat custom token
	customToken, err := f.client.CustomToken(ctx, uid)
	if err != nil {
		return "", err
	}
	
	// Exchange custom token menjadi ID token jika API key tersedia
	// ID token adalah yang biasa digunakan untuk autentikasi di API
	if f.apiKey != "" {
		idToken, err := f.exchangeCustomTokenForIDToken(customToken)
		if err != nil {
			return "", err
		}
		return idToken, nil
	}
	
	// Fallback ke custom token jika tidak bisa exchange
	return customToken, nil
}