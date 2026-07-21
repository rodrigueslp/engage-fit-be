package apiresponse

import "github.com/gin-gonic/gin"

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func Error(c *gin.Context, status int, code, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(status, ErrorBody{Code: code, Message: message, RequestID: stringValue(requestID)})
}

func AbortError(c *gin.Context, status int, code, message string) {
	requestID, _ := c.Get("request_id")
	c.AbortWithStatusJSON(status, ErrorBody{Code: code, Message: message, RequestID: stringValue(requestID)})
}

func stringValue(value any) string {
	if result, ok := value.(string); ok {
		return result
	}
	return ""
}
