package internal

import (
	"context"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/sanLimbu/todo-api/internal"
	envvar "github.com/sanLimbu/todo-api/internal/envar"
)

//NewRedis instantiates the Redis Client using configuration defined in env variables
func NewRedis(conf *envvar.Configuration) (*redis.Client, error) {

	host, err := conf.Get("REDIS_HOST")
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "conf.Get REDIS_HOST")
	}

	db, err := conf.Get("REDIS_DB")
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "conf.Get REDIS_DB")
	}

	dbi, _ := strconv.Atoi(db)

	rdb := redis.NewClient(&redis.Options{
		Addr: host,
		DB:   dbi,
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "rdb.Ping")
	}

	return rdb, nil

}
