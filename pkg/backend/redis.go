package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/garyburd/redigo/redis"
	log "github.com/sirupsen/logrus"
)

func newPool(server string, port int, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", server, port))
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

// RedisPool redis pool
type RedisPool struct {
	pool *redis.Pool
}

// RedisPoolService interface for getting Redis connection from pool or test mock
type RedisPoolService interface {
	GetConn() redis.Conn
}

// GetConn helper function: get Redis connection from pool; override in test
func (rp *RedisPool) GetConn() redis.Conn {
	return rp.pool.Get()
}

// RedisStore in memory trigger map store
type RedisStore struct {
	redisPool   RedisPoolService
	pipelineSvc codefresh.PipelineService
}

func getTriggerKey(id string) string {
	// set * for empty id
	if id == "" {
		id = "*"
	}
	if strings.HasPrefix(id, "trigger:") {
		return id
	}
	return fmt.Sprintf("trigger:%s", id)
}

// NewRedisStore create new Redis DB for storing trigger map
func NewRedisStore(server string, port int, password string, pipelineSvc codefresh.PipelineService) model.TriggerService {
	return &RedisStore{&RedisPool{newPool(server, port, password)}, pipelineSvc}
}

// List get list of defined triggers
func (r *RedisStore) List(filter string) ([]model.Trigger, error) {
	con := r.redisPool.GetConn()
	log.Debug("Getting all triggers ...")
	keys, err := redis.Values(con.Do("KEYS", getTriggerKey(filter)))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Iterate through all trigger keys and get triggers
	triggers := []model.Trigger{}
	for _, k := range keys {
		trigger, err := r.Get(string(k.([]uint8)))
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if !trigger.IsEmpty() {
			triggers = append(triggers, trigger)
		}
	}
	return triggers, nil
}

// Get trigger by key
func (r *RedisStore) Get(id string) (model.Trigger, error) {
	con := r.redisPool.GetConn()
	log.Debugf("Getting trigger %s ...", id)
	// get secret from String
	secret, err := redis.String(con.Do("GET", id))
	if err != nil {
		log.Error(err)
		return model.EmptyTrigger, err
	}
	// get pipelines from Set
	pipelines, err := redis.Values(con.Do("SMEMBERS", getTriggerKey(id)))
	if err != nil {
		log.Error(err)
		return model.EmptyTrigger, err
	}
	var trigger model.Trigger
	if len(pipelines) > 0 {
		trigger.Event = strings.TrimPrefix(id, "trigger:")
		trigger.Secret = secret
	}
	for _, p := range pipelines {
		var pipeline model.Pipeline
		json.Unmarshal(p.([]byte), &pipeline)
		if err != nil {
			log.Error(err)
			return model.EmptyTrigger, err
		}
		trigger.Pipelines = append(trigger.Pipelines, pipeline)
	}
	return trigger, nil
}

// Add new trigger {Event, Secret, Pipelines}
func (r *RedisStore) Add(trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	log.Debugf("Adding/Updating trigger %s ...", trigger.Event)
	// add secret to Redis String
	_, err := con.Do("SET", trigger.Event, trigger.Secret)
	if err != nil {
		log.Error(err)
		return err
	}
	// add pipelines to Redis Set
	for _, v := range trigger.Pipelines {
		pipeline, err := json.Marshal(v)
		if err != nil {
			log.Error(err)
			return err
		}
		log.Debugf("trigger '%s' <- '%s' pipeline \n", trigger.Event, pipeline)
		_, err = con.Do("SADD", getTriggerKey(trigger.Event), pipeline)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// Delete trigger by id
func (r *RedisStore) Delete(id string) error {
	con := r.redisPool.GetConn()
	log.Debugf("Deleting trigger %s ...", id)
	// delete Redis String (secret)
	if _, err := con.Do("DEL", id); err != nil {
		log.Error(err)
		return err
	}
	// delete Redis Set
	if _, err := con.Do("DEL", getTriggerKey(id)); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// Update trigger
func (r *RedisStore) Update(t model.Trigger) error {
	return r.Add(t)
}

// Run trigger pipelines
func (r *RedisStore) Run(id string, vars map[string]string) error {
	trigger, err := r.Get(id)
	if err != nil {
		log.Error(err)
		return err
	}
	for _, p := range trigger.Pipelines {
		err = r.pipelineSvc.RunPipeline(p.Name, p.RepoOwner, p.RepoName, vars)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// CheckSecret check trigger secret
func (r *RedisStore) CheckSecret(id string, secret string) error {
	con := r.redisPool.GetConn()
	log.Debugf("Getting trigger %s ...", id)
	// get secret from String
	triggerSecret, err := redis.String(con.Do("GET", id))
	if err != nil {
		log.Error(err)
		return err
	}
	if triggerSecret != secret {
		return errors.New("invalid secret")
	}
	return nil
}
