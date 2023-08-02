package mock

import "github.com/gin-gonic/gin"

type API interface {
	GetMockResponse(c *gin.Context)
}
