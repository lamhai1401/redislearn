package client

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

type RedisCacheRepo interface {
	FindItem(ctx context.Context, key string) ([]byte, error)
	SaveItem(ctx context.Context, key string, bytes []byte, ttl *time.Duration) error
	RemoveItem(ctx context.Context, key string) error

	FindKeys(ctx context.Context, keyword string, cursor uint64, match int64) ([]string, error)
	DeleteWithKeyword(ctx context.Context, keyword string, cursor uint64, count int64) error
	DeleteWithKeys(ctx context.Context, keys []string) error
	DeleteWithoutTTL(ctx context.Context, keyword string, cursor uint64, count int64) error
}

type redisRepo struct {
	log         *log.Helper
	redisClient *redis.Client
}

func NewRedisRepo(config *redis.Options, logger log.Logger) (RedisCacheRepo, func(), error) {
	rdb := redis.NewClient(config)

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, nil, err
	}

	result := &redisRepo{
		redisClient: rdb,
		log:         log.NewHelper(log.With(logger, "use-case", "cache", "cache-db", config.ClientName)),
	}

	return result, func() {
		rdb.Close()
	}, nil
}

func (r *redisRepo) FindItem(ctx context.Context, key string) ([]byte, error) {
	return r.redisGet(ctx, r.redisClient, key)
}

func (r *redisRepo) SaveItem(ctx context.Context, key string, bytes []byte, ttl *time.Duration) error {
	return r.redisSet(ctx, r.redisClient, key, bytes, ttl)
}

func (r *redisRepo) RemoveItem(ctx context.Context, key string) error {
	return r.redisRemove(ctx, r.redisClient, key)
}

func (r *redisRepo) FindKeys(ctx context.Context, keyword string, cursor uint64, count int64) ([]string, error) {
	var (
		items = make([]string, 0)
		iter  = r.redisClient.Scan(ctx, cursor, keyword, count).Iterator()
	)

	for iter.Next(ctx) {
		items = append(items, iter.Val())
	}

	if err := iter.Err(); err != nil {
		r.log.Error(err)
		return nil, err
	}

	return items, nil
}

func (r *redisRepo) DeleteWithKeyword(ctx context.Context, keyword string, cursor uint64, count int64) error {
	iter := r.redisClient.Scan(ctx, cursor, keyword, count).Iterator()
	for iter.Next(ctx) {
		if err := r.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			r.log.Error(err)
		}
	}

	if err := iter.Err(); err != nil {
		r.log.Error(err)
	}

	return nil
}

func (r *redisRepo) DeleteWithKeys(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := r.redisClient.Del(ctx, key).Err(); err != nil {
			r.log.Error(err)
		}
	}

	return nil
}

func (r *redisRepo) DeleteWithoutTTL(ctx context.Context, keyword string, cursor uint64, count int64) error {
	iter := r.redisClient.Scan(ctx, cursor, keyword, count).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		d, err := r.redisClient.TTL(ctx, key).Result()
		if err != nil {
			r.log.Error(err)
		} else if d == -1 {
			if err := r.redisClient.Del(ctx, key).Err(); err != nil {
				r.log.Error(err)
			}
		}
	}

	if err := iter.Err(); err != nil {
		r.log.Error(err)
	}

	return nil
}

func (r *redisRepo) redisSet(ctx context.Context, c *redis.Client, key string, value interface{}, ttl *time.Duration) error {
	if err := c.Set(ctx, key, value, *ttl).Err(); err != nil {
		r.log.Error(err)
		return err
	}
	return nil
}

func (r *redisRepo) redisRemove(ctx context.Context, c *redis.Client, key string) error {
	if err := c.Del(ctx, key).Err(); err != nil {
		r.log.Error(err)
		return err
	}
	return nil
}

func (r *redisRepo) redisGet(ctx context.Context, c *redis.Client, key string) ([]byte, error) {
	bytes, err := c.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}

	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	return bytes, nil
}
