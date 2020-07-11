package sesscache

import (
	"github.com/go-redis/redis"
	"github.com/jack0liu/conf"
	"time"
)

var re *redis.Client

func InitRedis() {
	redisAddr := conf.GetStringWithDefault("redis_addr", "127.0.0.1:6379")
	re = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err := re.Ping().Result()
	if err != nil {
		panic(err.Error())
	}
}

func Set(key, value string) {
	re.Set(key, value, time.Hour*24)
}

func SetWithExpired(key, value string, expiration time.Duration) {
	re.Set(key, value, expiration)
}

func SetWithNoExpired(key, value string) {
	re.Set(key, value, 0)
}

func Get(key string) string {
	result, err := re.Get(key).Result()
	if err != nil {
		//logs.Info("get key(%s) fail:%s", key, err.Error())
		return ""
	}
	return result
}

func Del(key string) {
	re.Del(key).Result()
}

func TouchWithExpired(key string, expiration time.Duration) {
	re.Expire(key, expiration)
}
