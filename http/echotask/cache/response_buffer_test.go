package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResponseBuffer(t *testing.T) {
	t.Parallel()
	writer := httptest.NewRecorder()
	rb := newResponseBuffer(writer)

	assert.NotNil(t, rb, "newResponseBuffer() should not return nil")
	assert.Equal(t, writer, rb.writer, "Writer should be initialized correctly")
	assert.NotNil(t, rb.body, "Body buffer should be initialized")
}

func TestResponseBufferHeader(t *testing.T) {
	t.Parallel()
	writer := httptest.NewRecorder()
	rb := newResponseBuffer(writer)

	expectedHeader := writer.Header()
	actualHeader := rb.Header()

	assert.Equal(t, expectedHeader, actualHeader, "Header() should return the underlying writer's header")
}

func TestResponseBufferWrite(t *testing.T) {
	t.Parallel()
	writer := httptest.NewRecorder()
	rb := newResponseBuffer(writer)

	data := []byte("Hello, World!")
	n, err := rb.Write(data)

	assert.NoError(t, err, "Write() should not return an error")
	assert.Equal(t, len(data), n, "Write() should return the correct number of bytes written")
	assert.Equal(t, data, rb.body.Bytes(), "Write() should capture the data in the buffer")
	assert.Equal(t, string(data), writer.Body.String(), "Write() should write data to the underlying writer")
}

func TestResponseBufferWriteHeader(t *testing.T) {
	t.Parallel()
	writer := httptest.NewRecorder()
	rb := newResponseBuffer(writer)

	statusCode := http.StatusCreated
	rb.WriteHeader(statusCode)

	assert.Equal(t, statusCode, writer.Code, "WriteHeader() should set the correct status code on the underlying writer")
}
