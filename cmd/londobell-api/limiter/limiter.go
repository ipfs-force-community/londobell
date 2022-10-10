package limiter

import (
	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
)

type LimiterIface interface {
	Key(c *gin.Context) string
	GetBucket(key string) (*ratelimit.Bucket, bool)
	AddBucketsByUri(uri string, fillInterval, capacity, quantum int64) LimiterIface
	AddBucketByConf() LimiterIface
}

type Limiter struct {
	limiterBuckets map[string]*ratelimit.Bucket
}
