package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"finish-line/api"
)

// RouteRegistrar lets each module contribute its routes without the server
// knowing module internals.
type RouteRegistrar interface {
	RegisterRoutes(r gin.IRouter)
}

// Modules groups route registrars by whether they require authentication.
// Public modules (login, refresh) are reachable without a token; protected
// ones sit behind the auth middleware.
type Modules struct {
	Public    []RouteRegistrar
	Protected []RouteRegistrar
}

func New(logger *slog.Logger, db *gorm.DB, authMW gin.HandlerFunc, modules Modules) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(logger))

	// Operational and meta endpoints live at the root, outside the versioned
	// API: health is for infrastructure probes and docs describe the API
	// rather than being part of it. Neither should move when the API version
	// changes.
	r.GET("/health", handleHealth(db))
	r.GET("/openapi.yaml", handleSpec())
	r.GET("/docs", handleDocs())

	// Business resources are versioned. Modules register their routes onto
	// the /api/v1 group without knowing the prefix exists.
	v1 := r.Group("/api/v1")

	public := v1.Group("")
	for _, m := range modules.Public {
		m.RegisterRoutes(public)
	}

	protected := v1.Group("")
	protected.Use(authMW)
	for _, m := range modules.Protected {
		m.RegisterRoutes(protected)
	}

	return r
}

func handleHealth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err == nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			err = sqlDB.PingContext(ctx)
		}
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func handleSpec() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", api.Spec)
	}
}

func handleDocs() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docsHTML))
	}
}

const docsHTML = `<!doctype html>
<html>
  <head>
    <title>FinishLine API</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script id="api-reference" data-url="/openapi.yaml"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start),
		)
	}
}
