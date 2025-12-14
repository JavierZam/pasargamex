package firebase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
)

type FirebaseAuthClient struct {
	client *auth.Client
	apiKey string
}

func NewFirebaseAuthClient(client *auth.Client, apiKey string) *FirebaseAuthClient {
	return &FirebaseAuthClient{
		client: client,
		apiKey: apiKey,
	}
}

func (f *FirebaseAuthClient) CreateUser(ctx context.Context, email, password, displayName string) (string, error) {
	log.Printf("Creating Firebase Auth user with email: %s", email)

	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		DisplayName(displayName)

	user, err := f.client.CreateUser(ctx, params)
	if err != nil {
		log.Printf("Firebase Auth error type: %T", err)
		log.Printf("Firebase Auth error detail: %v", err)

		return "", err
	}

	log.Printf("Firebase Auth user created successfully with UID: %s", user.UID)
	return user.UID, nil
}

func (f *FirebaseAuthClient) VerifyToken(ctx context.Context, token string) (string, error) {
	result, err := f.client.VerifyIDToken(ctx, token)
	if err != nil {
		return "", err
	}

	return result.UID, nil
}

func (f *FirebaseAuthClient) GenerateToken(ctx context.Context, uid string) (string, error) {
	customToken, err := f.client.CustomToken(ctx, uid)
	if err != nil {
		return "", err
	}

	if f.apiKey != "" {
		idToken, err := f.exchangeCustomTokenForIDToken(customToken)
		if err != nil {
			log.Printf("Failed to exchange custom token for ID token: %v", err)
			return customToken, nil
		}
		log.Printf("Successfully exchanged custom token for ID token")
		return idToken, nil
	}

	return customToken, nil
}

func (f *FirebaseAuthClient) exchangeCustomTokenForIDToken(customToken string) (string, error) {
	if f.apiKey == "" {
		return "", fmt.Errorf("Firebase API key is not set")
	}

	log.Printf("Exchanging custom token for ID token")
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithCustomToken?key=%s", f.apiKey)

	reqBody := fmt.Sprintf(`{"token":"%s","returnSecureToken":true}`, customToken)

	req, err := http.NewRequest("POST", url, strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("firebase auth API error: %s", string(body))
	}

	var result struct {
		IDToken string `json:"idToken"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.IDToken, nil
}

func (f *FirebaseAuthClient) SignInWithEmailPassword(email, password string) (string, error) {
	if f.apiKey == "" {
		log.Printf("Firebase API key is not set")
		return "", fmt.Errorf("Firebase API key is not set")
	}

	log.Printf("Signing in with email/password for: %s", email)
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", f.apiKey)

	reqBody := fmt.Sprintf(`{"email":"%s","password":"%s","returnSecureToken":true}`, email, password)

	req, err := http.NewRequest("POST", url, strings.NewReader(reqBody))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return "", err
	}

	log.Printf("Firebase Auth response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Firebase auth API error: %s", string(body))
		return "", fmt.Errorf("firebase auth API error: %s", string(body))
	}

	var result struct {
		IDToken string `json:"idToken"`
		LocalID string `json:"localId"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error parsing response: %v", err)
		return "", err
	}

	log.Printf("Successfully signed in user: %s with ID: %s", email, result.LocalID)
	return result.IDToken, nil
}

func (f *FirebaseAuthClient) UpdateUserPassword(ctx context.Context, uid, newPassword string) error {
	params := (&auth.UserToUpdate{}).
		Password(newPassword)

	_, err := f.client.UpdateUser(ctx, uid, params)
	if err != nil {
		return err
	}

	return nil
}

func (f *FirebaseAuthClient) TestConnection(ctx context.Context) error {
	_, err := f.client.GetUser(ctx, "non-existent-uid")
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			return nil
		}
		return err
	}
	return nil
}

func (f *FirebaseAuthClient) SignInWithEmailPasswordWithRefresh(email, password string) (string, string, error) {
	if f.apiKey == "" {
		return "", "", fmt.Errorf("Firebase API key is not set")
	}

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", f.apiKey)

	reqBody := fmt.Sprintf(`{"email":"%s","password":"%s","returnSecureToken":true}`, email, password)

	req, err := http.NewRequest("POST", url, strings.NewReader(reqBody))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("firebase auth API error: %s", string(body))
	}

	var result struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		LocalID      string `json:"localId"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	return result.IDToken, result.RefreshToken, nil
}

func (f *FirebaseAuthClient) RefreshIdToken(refreshToken string) (string, string, error) {
	if f.apiKey == "" {
		return "", "", fmt.Errorf("Firebase API key is not set")
	}

	apiURL := fmt.Sprintf("https://securetoken.googleapis.com/v1/token?key=%s", f.apiKey)

	data := fmt.Sprintf("grant_type=refresh_token&refresh_token=%s", refreshToken)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	log.Printf("Refresh token response: Status=%d, Body=%s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("firebase refresh token API error: %s", string(respBody))
	}

	var result struct {
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("error parsing response: %v", err)
	}

	return result.IDToken, result.RefreshToken, nil
}

func (f *FirebaseAuthClient) GenerateDevTokenPair(ctx context.Context, email string) (string, string, error) {
	user, err := f.client.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("error getting user: %v", err)
	}

	customToken, err := f.client.CustomToken(ctx, user.UID)
	if err != nil {
		return "", "", fmt.Errorf("error creating custom token: %v", err)
	}

	if f.apiKey == "" {
		return customToken, "", fmt.Errorf("firebase API key not available, cannot exchange for ID token")
	}

	reqData := struct {
		Token             string `json:"token"`
		ReturnSecureToken bool   `json:"returnSecureToken"`
	}{
		Token:             customToken,
		ReturnSecureToken: true,
	}

	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		return "", "", fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithCustomToken?key=%s", f.apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", "", fmt.Errorf("error exchanging token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", "", fmt.Errorf("error response from Firebase: %s", string(bodyBytes))
	}

	var result struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("error decoding response: %v", err)
	}

	return result.IDToken, result.RefreshToken, nil
}

// GetUserProfile gets Firebase Auth user profile data including OAuth profile info
func (f *FirebaseAuthClient) GetUserProfile(ctx context.Context, uid string) (*auth.UserRecord, error) {
	user, err := f.client.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %v", err)
	}
	return user, nil
}

// CreateOrUpdateUserFromFirebase creates or updates user with Firebase profile data
func (f *FirebaseAuthClient) CreateOrUpdateUserFromFirebase(ctx context.Context, uid string) (*auth.UserRecord, error) {
	user, err := f.client.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get Firebase user: %v", err)
	}
	
	log.Printf("Firebase user profile - UID: %s, Email: %s, DisplayName: %s, PhotoURL: %s", 
		user.UID, user.Email, user.DisplayName, user.PhotoURL)
	
	return user, nil
}
