package firebase

import (
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
	apiKey string // Tambahkan API key Firebase
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

// Fungsi ini tetap ada untuk kompatibilitas dengan kode yang sudah ada
func (f *FirebaseAuthClient) GenerateToken(ctx context.Context, uid string) (string, error) {
	customToken, err := f.client.CustomToken(ctx, uid)
	if err != nil {
		return "", err
	}
	
	// Jika apiKey tersedia, exchange ke ID token
	if f.apiKey != "" {
		idToken, err := f.exchangeCustomTokenForIDToken(customToken)
		if err != nil {
			log.Printf("Failed to exchange custom token for ID token: %v", err)
			// Fallback ke custom token jika gagal
			return customToken, nil
		}
		log.Printf("Successfully exchanged custom token for ID token")
		return idToken, nil
	}
	
	return customToken, nil
}

// Metode baru untuk exchange custom token ke ID token
func (f *FirebaseAuthClient) exchangeCustomTokenForIDToken(customToken string) (string, error) {
	if f.apiKey == "" {
		return "", fmt.Errorf("Firebase API key is not set")
	}

	log.Printf("Exchanging custom token for ID token")
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithCustomToken?key=%s", f.apiKey)
	
	// Prepare request body
	reqBody := fmt.Sprintf(`{"token":"%s","returnSecureToken":true}`, customToken)
	
	// Send request
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
	
	// Parse response
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
    
    // Prepare request body
    reqBody := fmt.Sprintf(`{"email":"%s","password":"%s","returnSecureToken":true}`, email, password)
    
    // Send request
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
    
    // Log response status
    log.Printf("Firebase Auth response status: %d", resp.StatusCode)
    
    if resp.StatusCode != http.StatusOK {
        log.Printf("Firebase auth API error: %s", string(body))
        return "", fmt.Errorf("firebase auth API error: %s", string(body))
    }
    
    // Parse response
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
	// Coba mengambil user yang tidak ada
	// Jika error adalah "user not found", berarti koneksi berhasil
	_, err := f.client.GetUser(ctx, "non-existent-uid")
	if err != nil {
		// Error user not found adalah expected dan menunjukkan koneksi berhasil
		if strings.Contains(err.Error(), "user not found") {
			return nil
		}
		// Error lain menunjukkan masalah koneksi
		return err
	}
	// Tidak ada error tapi user ditemukan - unlikely scenario
	return nil
}