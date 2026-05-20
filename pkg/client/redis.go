package client

import (
	"SREFlow/config"
	"context"
	"fmt"

	"github.com/go-redis/redis"
	"github.com/zeromicro/go-zero/core/logc"
)

func InitRedis() *redis.Client {
	conf := config.Application.Redis
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", conf.Host, conf.Port),
		Password: conf.Password,
		DB:       conf.Database,
	})

	_, err := client.Ping().Result()
	if err != nil {
		logc.Error(context.Background(), fmt.Sprintf("failed to connection redis, err: %s", err.Error()))
		panic(err)
	}

	logc.Info(context.Background(), "Cache 初始化完成!")
	return client
}
