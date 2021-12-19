package redis

import (
	"fmt"
	redisClient "github.com/gomodule/redigo/redis"
	"math/rand"
	"strconv"
	"time"
	"url-shortener/base62"
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
	conn := r.pool.Get()
	defer conn.Close()

	exists, err := redisClient.Bool(conn.Do("EXISTS", "Shortener:"+strconv.FormatUint(id, 10)))
	if err != nil {
		return true
	}
	return exists
}

func (r *redis) Save(url string, expires time.Time) (string, error) {
	conn := r.pool.Get()
	defer conn.Close()

	var id uint64

	for used := true; used; used = r.isUsed(id) {
		id = rand.Uint64()
	}

	shortLink := storage.Item{
		Id:      id,
		URL:     url,
		Expires: expires.Format("2006-01-02 15:04:05.728046+0300 EEST"),
	}

	_, err := conn.Do("HMSET",
		redisClient.Args{"Shortener:" + strconv.FormatUint(id, 10)}.AddFlat(shortLink)...)
	if err != nil {
		return "", err
	}

	_, err = conn.Do("EXPIREAT", "Shortener"+strconv.FormatUint(id, 10), expires.Unix())
	if err != nil {
		return "", err
	}

	return base62.Encode(id), nil
}

func (r *redis) Load(code string) (string, error) {
	conn := r.pool.Get()
	defer conn.Close()

	decodedId, err := base62.Decode(code)
	if err != nil {
		return "", err
	}

	urlString, err := redisClient.String(conn.Do("HGET", "Shortener:"+strconv.FormatUint(decodedId, 10), "url"))
	if err != nil {
		return "", err
	} else if len(urlString) == 0 {
		return "", storage.ErrNoLink
	}

	_, err = conn.Do("HINCRBY", "Shortener:"+strconv.FormatUint(decodedId, 10), "visits", 1)
	return urlString, nil
}

func (r *redis) isAvailable(id uint64) bool {
	conn := r.pool.Get()
	defer conn.Close()

	exists, err := redisClient.Bool(conn.Do("EXISTS", "Shortener:"+strconv.FormatUint(id, 10)))
	if err != nil {
		return false
	}
	return !exists
}

func (r *redis) LoadInfo(code string) (*storage.Item, error) {
	conn := r.pool.Get()
	defer conn.Close()

	decodedId, err := base62.Decode(code)
	if err != nil {
		return nil, err
	}

	values, err := redisClient.Values(conn.Do("HGETALL", "Shortener:"+strconv.FormatUint(decodedId, 10)))
	if err != nil {
		return nil, err
	} else if len(values) == 0 {
		return nil, storage.ErrNoLink
	}
	var shortLink storage.Item
	err = redisClient.ScanStruct(values, &shortLink)
	if err != nil {
		return nil, err
	}

	return &shortLink, nil
}

func (r *redis) Close() error {
	return r.pool.Close()
}
