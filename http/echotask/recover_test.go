package echotask_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/zircuit-labs/zkr-go-common/http/echotask"
	"github.com/zircuit-labs/zkr-go-common/log"
)

var ErrTest = errors.New("test")

func TestRecoverPanic(t *testing.T) {
	t.Parallel()

	logger := log.NewTestLogger(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := echotask.Recover(logger)(echo.HandlerFunc(func(c echo.Context) error {
		panic("test")
	}))
	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecoverError(t *testing.T) {
	t.Parallel()

	logger := log.NewTestLogger(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := echotask.Recover(logger)(echo.HandlerFunc(func(c echo.Context) error {
		return ErrTest
	}))
	err := h(c)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrTest)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecoverNormal(t *testing.T) {
	t.Parallel()

	logger := log.NewTestLogger(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := echotask.Recover(logger)(echo.HandlerFunc(func(c echo.Context) error {
		return nil
	}))
	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
