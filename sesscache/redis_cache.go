package sesscache

import (
	"github.com/go-redis/redis"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
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

func Get(key string) string {
	result, err := re.Get(key).Result()
	if err != nil {
		logs.Error("get key(%s) err:%s", key, err.Error())
	}
	return result
}

func Del(key string) {
	re.Del(key).Result()
}

func Touch(key string) {
	re.Expire(key, time.Hour*24)
}