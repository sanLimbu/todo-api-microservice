package internal

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/sanLimbu/todo-api/internal"
	envvar "github.com/sanLimbu/todo-api/internal/envar"
)

func NewMemcached(conf *envvar.Configuration) (*memcache.Client, error) {

	host, err := conf.Get("MEMCACHED_HOST")
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "conf.Get MEMCACHED_HOST")
	}

	// Assuming environment variable contains only one server
	client := memcache.New(host)

	if err := client.Ping(); err != nil {

		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "ping")
	}

	client.Timeout = 100 * time.Millisecond
	client.MaxIdleConns = 100

	return client, nil

}
