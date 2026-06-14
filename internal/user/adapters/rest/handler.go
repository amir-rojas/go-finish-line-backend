package rest

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/common/httpx"
	"finish-line/internal/user/domain"
)

// UserService is the consumer-side contract this adapter needs from the
// application layer.
type UserService interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, error)
	ByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	ByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) error
}

type Handler struct {
	svc UserService
}

func NewHandler(svc UserService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r gin.IRouter) {
	users := r.Group("/users")
	users.POST("", h.create)
	users.GET("", h.list)
	// Static routes are registered before the :id param route so "me" is
	// never mistaken for an id.
	users.GET("/me", h.me)
	users.PUT("/me/password", h.changePassword)
	users.GET("/:id", h.byID)
}

func (h *Handler) create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}

	u, err := h.svc.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toResponse(u))
}

func (h *Handler) list(c *gin.Context) {
	if email, ok := c.GetQuery("email"); ok {
		h.listByEmail(c, email)
		return
	}

	users, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	out := make([]userResponse, 0, len(users))
	for i := range users {
		out = append(out, toResponse(&users[i]))
	}
	c.JSON(http.StatusOK, out)
}

// listByEmail keeps collection semantics: an email with no match is an
// empty result, not an error — 404 is reserved for /users/{id}.
func (h *Handler) listByEmail(c *gin.Context, email string) {
	u, err := h.svc.ByEmail(c.Request.Context(), email)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusOK, []userResponse{})
	case err != nil:
		httpx.RespondError(c, err)
	default:
		c.JSON(http.StatusOK, []userResponse{toResponse(u)})
	}
}

// me returns the currently authenticated user, identified by the access
// token rather than a path parameter.
func (h *Handler) me(c *gin.Context) {
	userID, ok := httpx.UserID(c)
	if !ok {
		httpx.Unauthorized(c, "not authenticated")
		return
	}

	u, err := h.svc.ByID(c.Request.Context(), userID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(u))
}

func (h *Handler) changePassword(c *gin.Context) {
	userID, ok := httpx.UserID(c)
	if !ok {
		httpx.Unauthorized(c, "not authenticated")
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		httpx.RespondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) byID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "invalid user id")
		return
	}

	u, err := h.svc.ByID(c.Request.Context(), id)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(u))
}
