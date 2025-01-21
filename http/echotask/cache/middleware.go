package cache

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type (
	// cacher is an interface that defines methods for interacting with a caching system.
	// It provides functionality to retrieve and store byte array content with a specific key and duration.
	cacher interface {
		Get(key string) ([]byte, bool)
		Set(key string, content []byte)
	}
)

// ResponseCacheMiddleware provides caching for GET requests, storing responses for a specified TTL using a caching system.
func ResponseCacheMiddleware(cacher cacher) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet {
				return next(c)
			}

			key := c.Request().URL.String()

			if cachedContent, found := cacher.Get(key); found {
				c.Response().Header().Set(echo.HeaderContentType, "application/json")
				_, err := c.Response().Write(cachedContent)
				return err
			}

			res := c.Response()
			buf := newResponseBuffer(res.Writer)
			res.Writer = buf

			err := next(c)
			if err != nil {
				return err
			}

			cacher.Set(key, buf.body.Bytes())
			return nil
		}
	}
}
