package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestResponseCacheMiddleware(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		requestMethod string
		requestPath   string
		cacheData     map[string][]byte
		nextHandler   echo.HandlerFunc
		expectedBody  string
		isCached      bool
	}{
		{
			name:          "GET cached hit",
			requestMethod: http.MethodGet,
			requestPath:   "/cached",
			cacheData:     map[string][]byte{"/cached": []byte(`{"result":"cached"}`)},
			nextHandler:   nil,
			expectedBody:  `{"result":"cached"}`,
			isCached:      true,
		},
		{
			name:          "GET cache miss",
			requestMethod: http.MethodGet,
			requestPath:   "/not-cached",
			cacheData:     map[string][]byte{},
			nextHandler:   func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"result": "calculated"}) },
			expectedBody:  `{"result":"calculated"}` + "\n",
			isCached:      false,
		},
		{
			name:          "POST no cache",
			requestMethod: http.MethodPost,
			requestPath:   "/post",
			cacheData:     map[string][]byte{},
			nextHandler:   func(c echo.Context) error { return c.JSON(http.StatusCreated, map[string]string{"status": "created"}) },
			expectedBody:  `{"status":"created"}` + "\n",
			isCached:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			req := httptest.NewRequest(tt.requestMethod, tt.requestPath, http.NoBody)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			cache := NewMemory(100, time.Second)
			for key, data := range tt.cacheData {
				cache.Set(key, data)
			}

			middleware := ResponseCacheMiddleware(cache)

			handler := middleware(tt.nextHandler)
			err := handler(c)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, rec.Body.String())

			if tt.requestMethod == http.MethodPost {
				_, found := cache.Get(tt.requestPath)
				assert.False(t, found, "POST requests should not be cached")
				return
			}

			if !tt.isCached && tt.nextHandler != nil {
				cachedContent, found := cache.Get(tt.requestPath)
				assert.True(t, found)
				assert.Equal(t, tt.expectedBody, string(cachedContent))
			}
		})
	}
}

func TestResponseCacheMiddlewareTwice(t *testing.T) {
	t.Parallel()
	e := echo.New()

	requestPath := "/twice"
	cache := NewMemory(100, time.Second)
	middleware := ResponseCacheMiddleware(cache)

	// First request - cache miss
	req1 := httptest.NewRequest(http.MethodGet, requestPath, http.NoBody)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	nextHandler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"result": "calculated"})
	}

	handler := middleware(nextHandler)
	err := handler(c1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec1.Code)
	assert.Equal(t, `{"result":"calculated"}`+"\n", rec1.Body.String())

	// Verify cache has been populated after the first call
	cachedContent, found := cache.Get(requestPath)
	assert.True(t, found)
	assert.Equal(t, `{"result":"calculated"}`+"\n", string(cachedContent))

	// Second request - cache hit
	req2 := httptest.NewRequest(http.MethodGet, requestPath, http.NoBody)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	err = handler(c2)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, `{"result":"calculated"}`+"\n", rec2.Body.String())

	// Verify no changes occurred to cached data
	cachedContentAfterSecondCall, foundAfterSecondCall := cache.Get(requestPath)
	assert.True(t, foundAfterSecondCall)
	assert.Equal(t, string(cachedContent), string(cachedContentAfterSecondCall))
}
