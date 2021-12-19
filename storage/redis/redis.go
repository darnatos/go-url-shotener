package redis

import (
	"fmt"
	redisClient "github.com/gomodule/redigo/redis"
	"time"
	"url-shortener/storage"
)

type redis struct {
	pool *redisClient.Pool
}

func New(host, port, password string) (storage.Service, error) {
	pool := &redisClient.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redisClient.Conn, error) {
			return redisClient.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
		},
	}
	return &redis{pool}, nil
}

func (r *redis) isUsed(id uint64) bool {
	return false
}

func (r *redis) Save(url string, expires time.Time) (string, error) {
	return "", nil
}

func (r *redis) Load(code string) (string, error) {
	return "", nil
}

func (r *redis) LoadInfo(code string) (*storage.Item, error) {
	return nil, nil
}

func (r *redis) Close() error {
	return r.pool.Close()
}
