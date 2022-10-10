package limiter

import (
	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
)

type Iface interface {
	Key(c *gin.Context) string
	GetBucket(key string) (*ratelimit.Bucket, bool)
	AddBucketsByUri(uri string, fillInterval, capacity, quantum int64) Iface
	AddBucketByConf() Iface
}

type Limiter struct {
	limiterBuckets map[string]*ratelimit.Bucket
}
