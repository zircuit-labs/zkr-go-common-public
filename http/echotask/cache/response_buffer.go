package cache

import (
	"bytes"
	"net/http"
)

type (
	// responseBuffer is a struct that captures HTTP response data and wraps an http.ResponseWriter.
	// It includes a buffer to hold the response body for further use or inspection.
	responseBuffer struct {
		writer http.ResponseWriter
		body   *bytes.Buffer
	}
)

func newResponseBuffer(w http.ResponseWriter) *responseBuffer {
	return &responseBuffer{
		writer: w,
		body:   new(bytes.Buffer),
	}
}

func (rb *responseBuffer) Header() http.Header {
	return rb.writer.Header()
}

func (rb *responseBuffer) Write(data []byte) (int, error) {
	rb.body.Write(data) // Capture the data
	return rb.writer.Write(data)
}

func (rb *responseBuffer) WriteHeader(statusCode int) {
	rb.writer.WriteHeader(statusCode)
}
