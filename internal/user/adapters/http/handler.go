package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/user/domain"
)

// UserService is the consumer-side contract this adapter needs from the
// application layer.
type UserService interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, error)
	ByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	ByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
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
	users.GET("/:id", h.byID)
}

func (h *Handler) create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	u, err := h.svc.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		respondError(c, err)
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
		respondError(c, err)
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
		respondError(c, err)
	default:
		c.JSON(http.StatusOK, []userResponse{toResponse(u)})
	}
}

func (h *Handler) byID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid user id"})
		return
	}

	u, err := h.svc.ByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(u))
}

// respondError maps domain errors to HTTP status codes. Internal errors are
// logged but never leaked to the client.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		c.JSON(http.StatusConflict, errorResponse{Error: domain.ErrEmailTaken.Error()})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: domain.ErrNotFound.Error()})
	case isValidationError(err):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		slog.Error("unhandled error in user handler", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrNameRequired) ||
		errors.Is(err, domain.ErrEmailInvalid) ||
		errors.Is(err, domain.ErrPasswordTooShort) ||
		errors.Is(err, domain.ErrPasswordTooLong)
}
