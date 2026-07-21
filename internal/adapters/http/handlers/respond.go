package handlers

import (
	"errors"
	"net/http"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func notImplemented(c *gin.Context, resource string) {
	_ = resource
	apiresponse.Error(c, http.StatusNotImplemented, "not_implemented", "resource not implemented")
}

func respondError(c *gin.Context, err error) {
	_ = c.Error(err)
	status := http.StatusInternalServerError
	message := "internal server error"

	if errors.Is(err, gorm.ErrRecordNotFound) {
		status = http.StatusNotFound
		message = "resource not found"
	}

	code := "internal_error"
	if status == http.StatusNotFound {
		code = "not_found"
	}
	apiresponse.Error(c, status, code, message)
}

func respondBadRequest(c *gin.Context) {
	apiresponse.Error(c, http.StatusBadRequest, "invalid_request", "invalid request")
}

func respondUnauthorized(c *gin.Context) {
	apiresponse.Error(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
}

func respondPublicError(c *gin.Context, status int, code, message string) {
	apiresponse.Error(c, status, code, message)
}
