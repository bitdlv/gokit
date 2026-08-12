package cache

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

type Namespace string

const (
	PATROL_PREFIX Namespace = "patrol"
)

func MustRdsClient(opt *redis.Options) *redis.Client {
	rdb := redis.NewClient(opt)
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		panic(err)
	}
	return rdb
}

func Keys(ns Namespace, keys ...string) string {
	if len(keys) == 0 {
		return string(ns)
	}
	msg := make([]string, 0, len(keys)+1)
	msg = append(msg, string(ns))
	msg = append(msg, keys...)
	return strings.Join(msg, ":")
}
