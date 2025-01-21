package healthcheck

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

//go:generate mockgen -source handler.go -destination mock_handler.go -package healthcheck

type (
	GetHealthCheck struct {
		checker Checker
	}

	Checker interface {
		HealthCheck(ctx context.Context) error
	}
)

func New(checker Checker) *GetHealthCheck {
	return &GetHealthCheck{checker: checker}
}

func (g GetHealthCheck) Handle(c echo.Context) error {
	if err := g.checker.HealthCheck(c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	resp := NewHealthCheck(time.Now().UTC().String())

	return c.JSON(http.StatusOK, resp)
}
