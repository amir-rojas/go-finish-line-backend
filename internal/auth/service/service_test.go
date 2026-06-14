package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/auth/domain"
	"finish-line/internal/auth/service"
	userdomain "finish-line/internal/user/domain"
)

type fakeUserFinder struct {
	user *userdomain.User
	err  error
}

func (f *fakeUserFinder) ByEmail(_ context.Context, _ string) (*userdomain.User, error) {
	return f.user, f.err
}

type fakeVerifier struct {
	ok  bool
	err error
}

func (f *fakeVerifier) Verify(_, _ string) (bool, error) { return f.ok, f.err }

type fakeTokens struct {
	parseErr error
	userID   uuid.UUID
}

func (f *fakeTokens) Issue(uuid.UUID) (string, time.Time, error) {
	return "access-token", time.Now().Add(15 * time.Minute), nil
}

func (f *fakeTokens) Parse(string) (domain.Claims, error) {
	if f.parseErr != nil {
		return domain.Claims{}, f.parseErr
	}
	return domain.Claims{UserID: f.userID}, nil
}

type fakeRefreshRepo struct {
	stored        *domain.RefreshToken
	byHash        *domain.RefreshToken
	byHashErr     error
	rotatedID     uuid.UUID
	revokedFamily uuid.UUID
}

func (r *fakeRefreshRepo) Create(_ context.Context, t *domain.RefreshToken) error {
	r.stored = t
	return nil
}

func (r *fakeRefreshRepo) ByTokenHash(_ context.Context, _ string) (*domain.RefreshToken, error) {
	if r.byHashErr != nil {
		return nil, r.byHashErr
	}
	return r.byHash, nil
}

func (r *fakeRefreshRepo) MarkRotated(_ context.Context, id uuid.UUID) error {
	r.rotatedID = id
	return nil
}

func (r *fakeRefreshRepo) RevokeFamily(_ context.Context, familyID uuid.UUID) error {
	r.revokedFamily = familyID
	return nil
}

func validUser() *userdomain.User {
	return &userdomain.User{ID: uuid.New(), Email: "admin@finishline.dev", PasswordHash: "stored-hash"}
}

func TestLogin(t *testing.T) {
	t.Run("valid credentials issue a token pair and store a refresh token", func(t *testing.T) {
		repo := &fakeRefreshRepo{}
		svc := service.New(&fakeUserFinder{user: validUser()}, &fakeVerifier{ok: true}, &fakeTokens{}, repo, time.Hour)

		pair, err := svc.Login(context.Background(), "admin@finishline.dev", "admin.123")
		if err != nil {
			t.Fatalf("Login() unexpected error: %v", err)
		}
		if pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Error("Login() returned an empty token")
		}
		if repo.stored == nil {
			t.Fatal("Login() did not store a refresh token")
		}
		if repo.stored.TokenHash == pair.RefreshToken {
			t.Error("stored the raw refresh token instead of its hash")
		}
		if repo.stored.TokenHash != domain.HashToken(pair.RefreshToken) {
			t.Error("stored hash does not match the returned token")
		}
	})

	t.Run("wrong password fails as invalid credentials", func(t *testing.T) {
		svc := service.New(&fakeUserFinder{user: validUser()}, &fakeVerifier{ok: false}, &fakeTokens{}, &fakeRefreshRepo{}, time.Hour)

		_, err := svc.Login(context.Background(), "admin@finishline.dev", "wrong")
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("unknown email fails identically, without leaking", func(t *testing.T) {
		svc := service.New(&fakeUserFinder{err: userdomain.ErrNotFound}, &fakeVerifier{}, &fakeTokens{}, &fakeRefreshRepo{}, time.Hour)

		_, err := svc.Login(context.Background(), "ghost@finishline.dev", "whatever")
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestRefresh(t *testing.T) {
	t.Run("active token rotates: old marked, new stored in same family", func(t *testing.T) {
		family := uuid.New()
		active := &domain.RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			FamilyID:  family,
			ExpiresAt: time.Now().Add(time.Hour),
		}
		repo := &fakeRefreshRepo{byHash: active}
		svc := service.New(&fakeUserFinder{}, &fakeVerifier{}, &fakeTokens{}, repo, time.Hour)

		pair, err := svc.Refresh(context.Background(), "raw-token")
		if err != nil {
			t.Fatalf("Refresh() unexpected error: %v", err)
		}
		if repo.rotatedID != active.ID {
			t.Error("Refresh() did not mark the presented token as rotated")
		}
		if repo.stored == nil || repo.stored.FamilyID != family {
			t.Error("Refresh() did not store a new token in the same family")
		}
		if pair.RefreshToken == "" {
			t.Error("Refresh() returned an empty refresh token")
		}
	})

	t.Run("reused token revokes the whole family", func(t *testing.T) {
		rotatedAt := time.Now().Add(-time.Minute)
		family := uuid.New()
		used := &domain.RefreshToken{ID: uuid.New(), FamilyID: family, ExpiresAt: time.Now().Add(time.Hour), RotatedAt: &rotatedAt}
		repo := &fakeRefreshRepo{byHash: used}
		svc := service.New(&fakeUserFinder{}, &fakeVerifier{}, &fakeTokens{}, repo, time.Hour)

		_, err := svc.Refresh(context.Background(), "stolen-token")
		if !errors.Is(err, domain.ErrTokenReuse) {
			t.Errorf("Refresh() error = %v, want ErrTokenReuse", err)
		}
		if repo.revokedFamily != family {
			t.Error("Refresh() did not revoke the token family on reuse")
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		expired := &domain.RefreshToken{ID: uuid.New(), ExpiresAt: time.Now().Add(-time.Hour)}
		svc := service.New(&fakeUserFinder{}, &fakeVerifier{}, &fakeTokens{}, &fakeRefreshRepo{byHash: expired}, time.Hour)

		_, err := svc.Refresh(context.Background(), "old-token")
		if !errors.Is(err, domain.ErrInvalidToken) {
			t.Errorf("Refresh() error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("unknown token is rejected as invalid", func(t *testing.T) {
		svc := service.New(&fakeUserFinder{}, &fakeVerifier{}, &fakeTokens{}, &fakeRefreshRepo{byHashErr: domain.ErrRefreshTokenNotFound}, time.Hour)

		_, err := svc.Refresh(context.Background(), "nonsense")
		if !errors.Is(err, domain.ErrInvalidToken) {
			t.Errorf("Refresh() error = %v, want ErrInvalidToken", err)
		}
	})
}

func TestAuthenticate(t *testing.T) {
	t.Run("valid token returns the user id", func(t *testing.T) {
		want := uuid.New()
		svc := service.New(&fakeUserFinder{}, &fakeVerifier{}, &fakeTokens{userID: want}, &fakeRefreshRepo{}, time.Hour)

		got, err := svc.Authenticate("good-token")
		if err != nil {
			t.Fatalf("Authenticate() unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("Authenticate() = %v, want %v", got, want)
		}
	})

	t.Run("invalid token is rejected", func(t *testing.T) {
		svc := service.New(&fakeUserFinder{}, &fakeVerifier{}, &fakeTokens{parseErr: errors.New("bad signature")}, &fakeRefreshRepo{}, time.Hour)

		_, err := svc.Authenticate("tampered")
		if !errors.Is(err, domain.ErrInvalidToken) {
			t.Errorf("Authenticate() error = %v, want ErrInvalidToken", err)
		}
	})
}
