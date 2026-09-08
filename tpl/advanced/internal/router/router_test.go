package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"advanced/pkg/xserver"

	"github.com/gin-gonic/gin"
)

func TestHealthChecks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("healthz success", func(t *testing.T) {
		e := gin.New()
		HealthChecks(e)

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("ready success with checkers", func(t *testing.T) {
		e := gin.New()
		HealthChecks(e,
			NamedChecker{
				Name: "database",
				Check: func(ctx context.Context) error {
					return nil
				},
			},
			NamedChecker{
				Name: "redis",
				Check: func(ctx context.Context) error {
					return nil
				},
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != "READY" {
			t.Fatalf("expected READY, got %v", body["status"])
		}
	})

	t.Run("ready failure with error checker", func(t *testing.T) {
		e := gin.New()
		HealthChecks(e,
			NamedChecker{
				Name: "database",
				Check: func(ctx context.Context) error {
					return nil
				},
			},
			NamedChecker{
				Name: "redis",
				Check: func(ctx context.Context) error {
					return errors.New("connection timeout")
				},
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != "UNAVAILABLE" {
			t.Fatalf("expected UNAVAILABLE, got %v", body["status"])
		}
	})

	t.Run("ready draining state", func(t *testing.T) {
		e := gin.New()
		HealthChecks(e)

		xserver.SetDraining()

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 during draining, got %d", rec.Code)
		}
	})
}
