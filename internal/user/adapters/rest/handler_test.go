package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/common/httpx"
	"finish-line/internal/user/adapters/rest"
	"finish-line/internal/user/domain"
)

type fakeService struct {
	registerErr       error
	byIDErr           error
	byEmailErr        error
	changePasswordErr error
	users             []domain.User
}

func (s *fakeService) Register(_ context.Context, name, email, password string) (*domain.User, error) {
	if s.registerErr != nil {
		return nil, s.registerErr
	}
	return &domain.User{ID: uuid.New(), Name: name, Email: email, PasswordHash: "secret-hash"}, nil
}

func (s *fakeService) ByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if s.byIDErr != nil {
		return nil, s.byIDErr
	}
	return &domain.User{ID: id, Name: "Ana", Email: "ana@example.com", PasswordHash: "secret-hash"}, nil
}

func (s *fakeService) ByEmail(_ context.Context, email string) (*domain.User, error) {
	if s.byEmailErr != nil {
		return nil, s.byEmailErr
	}
	return &domain.User{ID: uuid.New(), Name: "Ana", Email: email, PasswordHash: "secret-hash"}, nil
}

func (s *fakeService) List(_ context.Context) ([]domain.User, error) {
	return s.users, nil
}

func (s *fakeService) ChangePassword(_ context.Context, _ uuid.UUID, _, _ string) error {
	return s.changePasswordErr
}

func setupRouter(svc *fakeService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rest.NewHandler(svc).RegisterRoutes(r)
	return r
}

// setupAuthedRouter injects an authenticated user id into the context, the
// way the real auth middleware would, so protected handlers can be tested.
func setupAuthedRouter(svc *fakeService, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		httpx.SetUserID(c, userID)
		c.Next()
	})
	rest.NewHandler(svc).RegisterRoutes(r)
	return r
}

func TestCreate(t *testing.T) {
	validBody := `{"name":"Ana","email":"ana@example.com","password":"secure-password"}`

	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "valid request", body: validBody, wantStatus: http.StatusCreated},
		{name: "malformed json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing fields", body: `{"name":"Ana"}`, wantStatus: http.StatusBadRequest},
		{name: "duplicate email", body: validBody, serviceErr: domain.ErrEmailTaken, wantStatus: http.StatusConflict},
		{name: "short password", body: validBody, serviceErr: domain.ErrPasswordTooShort, wantStatus: http.StatusBadRequest},
		{name: "invalid email", body: validBody, serviceErr: domain.ErrEmailInvalid, wantStatus: http.StatusBadRequest},
		{name: "unexpected error is masked", body: validBody, serviceErr: context.DeadlineExceeded, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter(&fakeService{registerErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if strings.Contains(rec.Body.String(), "secret-hash") {
				t.Error("response leaked the password hash")
			}
			if tt.wantStatus == http.StatusInternalServerError && strings.Contains(rec.Body.String(), "deadline") {
				t.Error("response leaked internal error details")
			}
		})
	}
}

func TestList(t *testing.T) {
	router := setupRouter(&fakeService{users: []domain.User{
		{ID: uuid.New(), Name: "Ana", Email: "ana@example.com", PasswordHash: "secret-hash"},
	}})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("response is not a JSON array: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if _, leaked := out[0]["password_hash"]; leaked {
		t.Error("response leaked the password hash field")
	}
}

func TestListByEmail(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		serviceErr error
		wantStatus int
		wantLen    int
	}{
		{name: "match returns one-element array", query: "?email=ana@example.com", wantStatus: http.StatusOK, wantLen: 1},
		{name: "no match returns empty array", query: "?email=ghost@example.com", serviceErr: domain.ErrNotFound, wantStatus: http.StatusOK, wantLen: 0},
		{name: "invalid email filter", query: "?email=not-an-email", serviceErr: domain.ErrEmailInvalid, wantStatus: http.StatusBadRequest, wantLen: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter(&fakeService{byEmailErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodGet, "/users"+tt.query, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantLen < 0 {
				return
			}

			var out []map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
				t.Fatalf("response is not a JSON array: %v", err)
			}
			if len(out) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(out), tt.wantLen)
			}
		})
	}
}

func TestMe(t *testing.T) {
	t.Run("returns the authenticated user", func(t *testing.T) {
		userID := uuid.New()
		router := setupAuthedRouter(&fakeService{}, userID)

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "secret-hash") {
			t.Error("response leaked the password hash")
		}
	})

	t.Run("401 when not authenticated", func(t *testing.T) {
		router := setupRouter(&fakeService{})

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestChangePassword(t *testing.T) {
	body := `{"current_password":"current-password","new_password":"brand-new-password"}`

	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "success returns 204", body: body, wantStatus: http.StatusNoContent},
		{name: "malformed body", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing fields", body: `{"current_password":"x"}`, wantStatus: http.StatusBadRequest},
		{name: "wrong current password", body: body, serviceErr: domain.ErrIncorrectPassword, wantStatus: http.StatusUnauthorized},
		{name: "new password too short", body: body, serviceErr: domain.ErrPasswordTooShort, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupAuthedRouter(&fakeService{changePasswordErr: tt.serviceErr}, uuid.New())

			req := httptest.NewRequest(http.MethodPut, "/users/me/password", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}

func TestByID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		serviceErr error
		wantStatus int
	}{
		{name: "found", path: "/users/" + uuid.NewString(), wantStatus: http.StatusOK},
		{name: "not found", path: "/users/" + uuid.NewString(), serviceErr: domain.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "invalid uuid", path: "/users/not-a-uuid", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter(&fakeService{byIDErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}
