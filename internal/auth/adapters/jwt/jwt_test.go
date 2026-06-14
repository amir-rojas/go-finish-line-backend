package jwt_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	authjwt "finish-line/internal/auth/adapters/jwt"
)

func TestIssueAndParse(t *testing.T) {
	svc := authjwt.New("test-secret", time.Hour)
	userID := uuid.New()

	token, expiresAt, err := svc.Issue(userID)
	if err != nil {
		t.Fatalf("Issue() unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("Issue() returned an empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Error("Issue() returned an already-expired token")
	}

	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("Parse() UserID = %v, want %v", claims.UserID, userID)
	}
}

func TestParseRejects(t *testing.T) {
	svc := authjwt.New("test-secret", time.Hour)

	t.Run("garbage token", func(t *testing.T) {
		if _, err := svc.Parse("not.a.jwt"); err == nil {
			t.Error("Parse() accepted a garbage token")
		}
	})

	t.Run("token signed with another secret", func(t *testing.T) {
		other := authjwt.New("different-secret", time.Hour)
		token, _, _ := other.Issue(uuid.New())
		if _, err := svc.Parse(token); err == nil {
			t.Error("Parse() accepted a token signed with a different secret")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		expiring := authjwt.New("test-secret", -time.Minute)
		token, _, _ := expiring.Issue(uuid.New())
		if _, err := svc.Parse(token); err == nil {
			t.Error("Parse() accepted an expired token")
		}
	})
}
