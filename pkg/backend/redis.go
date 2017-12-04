package backend

import (
	"time"

	"github.com/codefresh-io/cf-triggers/pkg/model"
	"github.com/garyburd/redigo/redis"
)

// RedisStore in memory trigger map store
type RedisStore struct {
	pool *redis.Pool
}

func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, nil
		},
	}
}

// NewRedisStore create new Redis DB for storing trigger map
func NewRedisStore(server, password string) model.TriggerService {
	return &RedisStore{newPool(server, password)}
}

// List get list of defined triggers
func (r *RedisStore) List() ([]model.Trigger, error) {
	con := r.pool.Get()
	con.Do("KEYS", "trigger:*")
	return nil, nil
}

func (r *RedisStore) Get(string) (model.Trigger, error) {
	return model.Trigger{}, nil
}

func (r *RedisStore) Add(model.Trigger) error {
	return nil
}

func (r *RedisStore) Delete(string) error {
	return nil
}

func (r *RedisStore) Update(model.Trigger) error {
	return nil
}
