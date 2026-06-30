package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func notImplemented(c *gin.Context, resource string) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"resource": resource,
		"status":   "not_implemented",
	})
}

func respondError(c *gin.Context, err error) {
	_ = c.Error(err)
	status := http.StatusInternalServerError
	message := "internal server error"

	if errors.Is(err, gorm.ErrRecordNotFound) {
		status = http.StatusNotFound
		message = "resource not found"
	}

	c.JSON(status, gin.H{"message": message})
}

func respondBadRequest(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
}

func respondUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
}
