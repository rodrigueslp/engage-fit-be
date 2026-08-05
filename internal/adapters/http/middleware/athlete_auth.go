package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/apiresponse"
	"boxengage/backend/internal/domain"
)

const (
	contextAthleteIDKey = "auth.athlete_id"
	contextAthleteKey   = "auth.athlete_context"
)

type AthleteSessionAuthenticator interface {
	Authenticate(ctx context.Context, token string) (*domain.AthleteContext, error)
}

func AthleteAuth(authenticator AthleteSessionAuthenticator, config SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, transport := tokenFromRequest(c, config)
		if token == "" {
			apiresponse.AbortError(c, http.StatusUnauthorized, "athlete_session_missing", "missing athlete session")
			return
		}
		athleteContext, err := authenticator.Authenticate(c.Request.Context(), token)
		if err != nil {
			apiresponse.AbortError(c, http.StatusUnauthorized, "athlete_session_invalid", "invalid athlete session")
			return
		}
		c.Set(contextAthleteIDKey, athleteContext.Account.ID)
		c.Set(contextAthleteKey, *athleteContext)
		c.Set(authTransportKey, transport)
		c.Next()
	}
}

func AthleteID(c *gin.Context) (domain.ID, error) {
	value, ok := c.Get(contextAthleteIDKey)
	if !ok {
		return "", errors.New("missing athlete id in context")
	}
	id, ok := value.(domain.ID)
	if !ok {
		return "", errors.New("invalid athlete id in context")
	}
	return id, nil
}

func AthleteContext(c *gin.Context) (domain.AthleteContext, error) {
	value, ok := c.Get(contextAthleteKey)
	if !ok {
		return domain.AthleteContext{}, errors.New("missing athlete context")
	}
	athleteContext, ok := value.(domain.AthleteContext)
	if !ok {
		return domain.AthleteContext{}, errors.New("invalid athlete context")
	}
	return athleteContext, nil
}
