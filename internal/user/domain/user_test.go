package domain_test

import (
	"errors"
	"testing"

	"finish-line/internal/user/domain"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		userName     string
		email        string
		passwordHash string
		wantErr      error
	}{
		{
			name:         "valid user",
			userName:     "Ana Pérez",
			email:        "ana@example.com",
			passwordHash: "some-hash",
			wantErr:      nil,
		},
		{
			name:         "empty name",
			userName:     "",
			email:        "ana@example.com",
			passwordHash: "some-hash",
			wantErr:      domain.ErrNameRequired,
		},
		{
			name:         "whitespace-only name",
			userName:     "   ",
			email:        "ana@example.com",
			passwordHash: "some-hash",
			wantErr:      domain.ErrNameRequired,
		},
		{
			name:         "invalid email",
			userName:     "Ana Pérez",
			email:        "not-an-email",
			passwordHash: "some-hash",
			wantErr:      domain.ErrEmailInvalid,
		},
		{
			name:         "missing password hash",
			userName:     "Ana Pérez",
			email:        "ana@example.com",
			passwordHash: "",
			wantErr:      domain.ErrPasswordHashRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := domain.New(tt.userName, tt.email, tt.passwordHash)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("New() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("New() unexpected error: %v", err)
			}
			if u.ID.String() == "00000000-0000-0000-0000-000000000000" {
				t.Error("New() did not assign an ID")
			}
			if u.CreatedAt.IsZero() {
				t.Error("New() did not assign CreatedAt")
			}
		})
	}
}

func TestNew_NormalizesEmail(t *testing.T) {
	u, err := domain.New("Ana", "  Ana.Perez@Example.COM ", "hash")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	if got, want := u.Email, "ana.perez@example.com"; got != want {
		t.Errorf("Email = %q, want %q", got, want)
	}
}

func TestNew_TrimsName(t *testing.T) {
	u, err := domain.New("  Ana  ", "ana@example.com", "hash")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	if got, want := u.Name, "Ana"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "valid password", password: "secure-password", wantErr: nil},
		{name: "exactly 8 chars", password: "12345678", wantErr: nil},
		{name: "too short", password: "1234567", wantErr: domain.ErrPasswordTooShort},
		{name: "empty", password: "", wantErr: domain.ErrPasswordTooShort},
		{name: "too long", password: string(make([]byte, 73)), wantErr: domain.ErrPasswordTooLong},
		{name: "exactly 72 chars", password: string(make([]byte, 72)), wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.ValidatePassword(tt.password); !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidatePassword() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
