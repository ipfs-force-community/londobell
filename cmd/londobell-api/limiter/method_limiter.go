package limiter

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
	"github.com/spf13/viper"
)

type URILimiter struct {
	*Limiter
	Rule *LimitConfRules
}

func NewUriLimiter() Iface {
	return &URILimiter{
		Limiter: &Limiter{
			limiterBuckets: make(map[string]*ratelimit.Bucket),
		},
	}
}

func (l *URILimiter) Key(c *gin.Context) string {
	uri := c.Request.RequestURI
	index := strings.Index(uri, "?")
	if index == -1 {
		return uri
	}
	return uri[:index]
}

func (l *URILimiter) GetBucket(key string) (*ratelimit.Bucket, bool) {
	fmt.Println(key, l.limiterBuckets)
	bucket, ok := l.limiterBuckets[key]
	return bucket, ok
}

func (l *URILimiter) AddBucketsByUri(uri string, fillInterval, capacity, quantum int64) Iface {
	bucket := ratelimit.NewBucketWithQuantum(time.Second*time.Duration(fillInterval), capacity, quantum)
	l.limiterBuckets[uri] = bucket
	return l
}

func (l *URILimiter) getConf() *LimitConfRules {
	once := sync.Once{}
	rule := &LimitConfRules{}
	once.Do(func() {
		vp := viper.New()
		vp.SetConfigFile("config.yaml")
		err := vp.ReadInConfig()
		if err != nil {
			log.Fatalln("read config.yaml error :", err)
		}
		errRule := vp.Unmarshal(&rule)
		if errRule != nil {
			log.Fatalln("unmarshal err :", errRule)
		}
	})
	return rule
}

func (l *URILimiter) AddBucketByConf() Iface {
	rule := l.getConf()
	for k, v := range rule.Rules {
		l.AddBucketsByUri(k, v.Interval, v.Capacity, v.Quantum)
	}
	return l
}
