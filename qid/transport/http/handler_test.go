package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeResponse_WithError(t *testing.T) {
	w := httptest.NewRecorder()
	resp := ErrorResponse{Err: "something went wrong"}
	encodeResponse(resp, w)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var got ErrorResponse
	json.NewDecoder(w.Body).Decode(&got)
	assert.Equal(t, "something went wrong", got.Err)
}

func TestEncodeResponse_WithoutError(t *testing.T) {
	w := httptest.NewRecorder()
	resp := ErrorResponse{Err: ""}
	encodeResponse(resp, w)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEncodeResponse_NonErrorer(t *testing.T) {
	w := httptest.NewRecorder()
	encodeResponse(struct{ Name string }{Name: "test"}, w)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var got ErrorResponse
	json.NewDecoder(w.Body).Decode(&got)
	assert.Contains(t, got.Err, "is not 'Errorer'")
}

func TestErrorResponse_Error(t *testing.T) {
	e := ErrorResponse{Err: "test error"}
	assert.Equal(t, "test error", e.Error())

	e = ErrorResponse{Err: ""}
	assert.Equal(t, "", e.Error())
}
