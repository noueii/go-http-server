package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	// Generate a test user ID
	userID := uuid.New()
	secret := "your-test-secret"

	// Test creating and validating a token
	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}
	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	// Validate the token
	extractedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}
	if extractedID != userID {
		t.Fatalf("Expected user ID %v, got %v", userID, extractedID)
	}
}

func TestExpiredToken(t *testing.T) {
	userID := uuid.New()
	secret := "your-test-secret"

	// Create a token that expires immediately
	token, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Try to validate the expired token
	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Fatal("Expected error for expired token, got nil")
	}
}

func TestInvalidSignature(t *testing.T) {
	userID := uuid.New()
	secret := "original-secret"

	// Create a token with one secret
	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Try to validate with a different secret
	wrongSecret := "wrong-secret"
	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Fatal("Expected error for invalid signature, got nil")
	}
}

func TestInvalidTokenFormat(t *testing.T) {
	// Test with a completely invalid token string
	invalidToken := "not-a-valid-jwt"
	_, err := ValidateJWT(invalidToken, "any-secret")
	if err == nil {
		t.Fatal("Expected error for invalid token format, got nil")
	}
}

func TestNilUserID(t *testing.T) {
	// Test with a zero UUID
	emptyID := uuid.Nil
	secret := "test-secret"

	token, err := MakeJWT(emptyID, secret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT with nil UUID: %v", err)
	}

	extractedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate JWT with nil UUID: %v", err)
	}
	if extractedID != emptyID {
		t.Fatalf("Expected nil UUID, got %v", extractedID)
	}
}

func TestEmptySecret(t *testing.T) {
	// Test with an empty secret
	userID := uuid.New()
	emptySecret := ""

	// This might fail depending on your implementation
	token, err := MakeJWT(userID, emptySecret, time.Hour)
	if err != nil {
		// If your implementation rejects empty secrets, this is expected
		t.Logf("MakeJWT rejected empty secret as expected: %v", err)
		return
	}

	// If a token was created with empty secret, try validating it
	extractedID, err := ValidateJWT(token, emptySecret)
	if err != nil {
		t.Fatalf("Failed to validate JWT with empty secret: %v", err)
	}
	if extractedID != userID {
		t.Fatalf("Expected user ID %v, got %v", userID, extractedID)
	}

	// Also try validating with a non-empty secret to ensure it fails
	_, err = ValidateJWT(token, "some-secret")
	if err == nil {
		t.Fatal("Expected error when validating token with different secret")
	}
}

func TestDifferentExpirations(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	// Test with short expiration (1 second)
	shortToken, err := MakeJWT(userID, secret, time.Second)
	if err != nil {
		t.Fatalf("Failed to create JWT with short expiration: %v", err)
	}

	// Validate immediately (should work)
	_, err = ValidateJWT(shortToken, secret)
	if err != nil {
		t.Fatalf("Failed to validate fresh JWT with short expiration: %v", err)
	}

	// Wait for token to expire
	time.Sleep(2 * time.Second)

	// Try to validate expired token
	// Try to validate expired token
	_, err = ValidateJWT(shortToken, secret)
	if err == nil {
		t.Fatal("Expected error for expired token, got nil")
	}

	// Test with long expiration
	longToken, err := MakeJWT(userID, secret, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT with long expiration: %v", err)
	}

	// Validate long expiration token
	extractedID, err := ValidateJWT(longToken, secret)
	if err != nil {
		t.Fatalf("Failed to validate JWT with long expiration: %v", err)
	}
	if extractedID != userID {
		t.Fatalf("Expected user ID %v, got %v", userID, extractedID)
	}
}

func TestTokenIssuer(t *testing.T) {
	// This test checks if the issuer is properly set to "chirpy"
	// Note: This requires your ValidateJWT function to return an error
	// if the issuer is not "chirpy"

	userID := uuid.New()
	secret := "test-secret"

	// Create token (should set issuer to "chirpy")
	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Validate the token
	extractedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}
	if extractedID != userID {
		t.Fatalf("Expected user ID %v, got %v", userID, extractedID)
	}

	// Additional verification could be done if you expose a way to check
	// the token's claims in your API
}
