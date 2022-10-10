package limiter

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func RateLimiter(l LimiterIface) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := l.Key(c)

		if bucket, ok := l.GetBucket(key); ok {
			count := bucket.TakeAvailable(1)
			if count == 0 {
				fmt.Println("rate limit")
				c.AbortWithStatusJSON(404, gin.H{
					"msg": "rate limit",
				})
				return
			}
		}
		c.Next()
	}
}
