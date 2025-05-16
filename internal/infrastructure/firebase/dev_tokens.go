package firebase

import (
	"context"
)

func (f *FirebaseAuthClient) GenerateLongLivedToken(ctx context.Context, uid string) (string, error) {

	customToken, err := f.client.CustomToken(ctx, uid)
	if err != nil {
		return "", err
	}

	if f.apiKey != "" {
		idToken, err := f.exchangeCustomTokenForIDToken(customToken)
		if err != nil {
			return "", err
		}
		return idToken, nil
	}

	return customToken, nil
}
