package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CorrelationIDMiddleware injects a unique X-Correlation-ID into the request context
// and the response header. This ID should be passed to downstream services (gRPC/HTTP)
// for distributed tracing.
func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Set for downstream use in the current service
		c.Set("X-Correlation-ID", correlationID)
		
		// Return it to the client
		c.Header("X-Correlation-ID", correlationID)

		c.Next()
	}
}
