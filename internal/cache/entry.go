package cache

import (
	"SREFlow/pkg/client"

	"github.com/go-redis/redis"
)

type (
	entry struct {
		redis *redis.Client
	}

	EntryInter interface {
		Redis() *redis.Client
	}
)

func NewEntryCache() EntryInter {
	return &entry{
		redis: client.InitRedis(),
	}
}

func (e entry) Redis() *redis.Client { return e.redis }
