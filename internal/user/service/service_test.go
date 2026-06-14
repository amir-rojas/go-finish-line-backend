package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"finish-line/internal/user/domain"
	"finish-line/internal/user/service"
)

type fakeRepo struct {
	users     map[string]*domain.User
	createErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: make(map[string]*domain.User)}
}

func (r *fakeRepo) Create(_ context.Context, u *domain.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	if _, exists := r.users[u.Email]; exists {
		return domain.ErrEmailTaken
	}
	r.users[u.Email] = u
	return nil
}

func (r *fakeRepo) ByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepo) ByEmail(_ context.Context, email string) (*domain.User, error) {
	if u, ok := r.users[email]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepo) List(_ context.Context) ([]domain.User, error) {
	out := make([]domain.User, 0, len(r.users))
	for _, u := range r.users {
		out = append(out, *u)
	}
	return out, nil
}

func (r *fakeRepo) UpdatePassword(_ context.Context, id uuid.UUID, passwordHash string) error {
	for _, u := range r.users {
		if u.ID == id {
			u.PasswordHash = passwordHash
			return nil
		}
	}
	return domain.ErrNotFound
}

type fakeSessionRevoker struct {
	revokedUserID uuid.UUID
	err           error
}

func (r *fakeSessionRevoker) RevokeAllSessions(_ context.Context, userID uuid.UUID) error {
	if r.err != nil {
		return r.err
	}
	r.revokedUserID = userID
	return nil
}

type fakeHasher struct {
	hashErr error
}

func (h *fakeHasher) Hash(plain string) (string, error) {
	if h.hashErr != nil {
		return "", h.hashErr
	}
	return "hashed:" + plain, nil
}

func (h *fakeHasher) Verify(plain, hash string) (bool, error) {
	return hash == "hashed:"+plain, nil
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		email    string
		password string
		setup    func(*fakeRepo, *fakeHasher)
		wantErr  error
	}{
		{
			name:     "valid registration",
			userName: "Ana",
			email:    "ana@example.com",
			password: "secure-password",
		},
		{
			name:     "password too short fails before hashing",
			userName: "Ana",
			email:    "ana@example.com",
			password: "short",
			wantErr:  domain.ErrPasswordTooShort,
		},
		{
			name:     "invalid email",
			userName: "Ana",
			email:    "nope",
			password: "secure-password",
			wantErr:  domain.ErrEmailInvalid,
		},
		{
			name:     "duplicate email surfaces ErrEmailTaken",
			userName: "Ana",
			email:    "ana@example.com",
			password: "secure-password",
			setup: func(r *fakeRepo, _ *fakeHasher) {
				existing, _ := domain.New("Ana", "ana@example.com", "hash")
				r.users[existing.Email] = existing
			},
			wantErr: domain.ErrEmailTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			hasher := &fakeHasher{}
			if tt.setup != nil {
				tt.setup(repo, hasher)
			}
			svc := service.New(repo, hasher, &fakeSessionRevoker{})

			u, err := svc.Register(context.Background(), tt.userName, tt.email, tt.password)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Register() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Register() unexpected error: %v", err)
			}
			if u.PasswordHash != "hashed:"+tt.password {
				t.Errorf("PasswordHash = %q, want hash of input password", u.PasswordHash)
			}
			if u.PasswordHash == tt.password {
				t.Error("password stored in plain text")
			}
			if _, err := repo.ByEmail(context.Background(), tt.email); err != nil {
				t.Errorf("user was not persisted: %v", err)
			}
		})
	}
}

func TestChangePassword(t *testing.T) {
	setup := func() (*service.Service, *fakeRepo, *fakeSessionRevoker, uuid.UUID) {
		repo := newFakeRepo()
		hasher := &fakeHasher{}
		revoker := &fakeSessionRevoker{}
		svc := service.New(repo, hasher, revoker)
		u, _ := svc.Register(context.Background(), "Ana", "ana@example.com", "current-password")
		return svc, repo, revoker, u.ID
	}

	t.Run("valid change updates the hash and revokes all sessions", func(t *testing.T) {
		svc, repo, revoker, id := setup()

		if err := svc.ChangePassword(context.Background(), id, "current-password", "brand-new-password"); err != nil {
			t.Fatalf("ChangePassword() unexpected error: %v", err)
		}
		u, _ := repo.ByID(context.Background(), id)
		if u.PasswordHash != "hashed:brand-new-password" {
			t.Errorf("password hash not updated, got %q", u.PasswordHash)
		}
		if revoker.revokedUserID != id {
			t.Error("ChangePassword() did not revoke the user's sessions")
		}
	})

	t.Run("wrong current password is rejected", func(t *testing.T) {
		svc, _, _, id := setup()

		err := svc.ChangePassword(context.Background(), id, "wrong-current", "brand-new-password")
		if !errors.Is(err, domain.ErrIncorrectPassword) {
			t.Errorf("ChangePassword() error = %v, want ErrIncorrectPassword", err)
		}
	})

	t.Run("new password must satisfy the policy", func(t *testing.T) {
		svc, _, _, id := setup()

		err := svc.ChangePassword(context.Background(), id, "current-password", "short")
		if !errors.Is(err, domain.ErrPasswordTooShort) {
			t.Errorf("ChangePassword() error = %v, want ErrPasswordTooShort", err)
		}
	})
}

func TestRegister_HasherFailure(t *testing.T) {
	repo := newFakeRepo()
	hasher := &fakeHasher{hashErr: errors.New("boom")}
	svc := service.New(repo, hasher, &fakeSessionRevoker{})

	_, err := svc.Register(context.Background(), "Ana", "ana@example.com", "secure-password")
	if err == nil {
		t.Fatal("Register() expected error when hasher fails")
	}
	if len(repo.users) != 0 {
		t.Error("user must not be persisted when hashing fails")
	}
}
