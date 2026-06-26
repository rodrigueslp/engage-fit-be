package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/domain"
)

const (
	contextUserIDKey = "auth.user_id"
	contextBoxIDKey  = "auth.box_id"
	contextRoleKey   = "auth.role"
)

func SetAuthContext(c *gin.Context, userID, boxID domain.ID, role domain.UserRole) {
	c.Set(contextUserIDKey, userID)
	c.Set(contextBoxIDKey, boxID)
	c.Set(contextRoleKey, role)
}

func UserID(c *gin.Context) (domain.ID, error) {
	value, ok := c.Get(contextUserIDKey)
	if !ok {
		return "", errors.New("missing user id in context")
	}
	id, ok := value.(domain.ID)
	if !ok {
		return "", errors.New("invalid user id in context")
	}
	return id, nil
}

func BoxID(c *gin.Context) (domain.ID, error) {
	value, ok := c.Get(contextBoxIDKey)
	if !ok {
		return "", errors.New("missing box id in context")
	}
	id, ok := value.(domain.ID)
	if !ok {
		return "", errors.New("invalid box id in context")
	}
	return id, nil
}

func Role(c *gin.Context) (domain.UserRole, error) {
	value, ok := c.Get(contextRoleKey)
	if !ok {
		return "", errors.New("missing role in context")
	}
	role, ok := value.(domain.UserRole)
	if !ok {
		return "", errors.New("invalid role in context")
	}
	return role, nil
}
