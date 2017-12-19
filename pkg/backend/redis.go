package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
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
	storeSvc    redisStoreInterface
}

// redisStoreInterface interfacing methods for better testability
type redisStoreInterface interface {
	storeTrigger(r *RedisStore, trigger model.Trigger) error
}

type redisStoreInternal struct {
}

// common code for RedisStore.Add() and RedisStore.Update()
func (redisStoreInternal) storeTrigger(r *RedisStore, trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	// generate random secret if required
	if trigger.Secret == model.GenerateKeyword {
		trigger.Secret = util.RandomString(16)
	}

	// add secret to Redis String
	_, err := con.Do("SET", trigger.Event, trigger.Secret)
	if err != nil {
		log.Error(err)
		return err
	}
	// add pipelines to Redis Set
	for _, v := range trigger.Pipelines {
		// check Codefresh pipeline existence
		err := r.pipelineSvc.CheckPipelineExist(v.Name, v.RepoOwner, v.RepoName)
		if err != nil {
			log.Error(err)
			return err
		}
		// marshal pipeline to JSON
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
	return &RedisStore{&RedisPool{newPool(server, port, password)}, pipelineSvc, &redisStoreInternal{}}
}

// List get list of defined triggers
func (r *RedisStore) List(filter string) ([]*model.Trigger, error) {
	con := r.redisPool.GetConn()
	log.Debug("Getting all triggers ...")
	keys, err := redis.Values(con.Do("KEYS", getTriggerKey(filter)))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Iterate through all trigger keys and get triggers
	triggers := []*model.Trigger{}
	for _, k := range keys {
		trigger, err := r.Get(string(k.([]uint8)))
		if err != nil {
			log.Error(err)
			return nil, err
		}
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

// Get trigger by key
func (r *RedisStore) Get(id string) (*model.Trigger, error) {
	con := r.redisPool.GetConn()
	// remove "trigger:" prefix
	id = strings.TrimPrefix(id, "trigger:")
	log.Debugf("Getting trigger %s ...", id)
	// get secret from String
	secret, err := redis.String(con.Do("GET", id))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return nil, err
	}
	// get pipelines from Set
	pipelines, err := redis.Values(con.Do("SMEMBERS", getTriggerKey(id)))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return nil, err
	} else if err == redis.ErrNil {
		return nil, model.ErrTriggerNotFound
	}
	trigger := new(model.Trigger)
	if len(pipelines) > 0 {
		trigger.Event = id
		trigger.Secret = secret
	}
	for _, p := range pipelines {
		var pipeline model.Pipeline
		json.Unmarshal(p.([]byte), &pipeline)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		trigger.Pipelines = append(trigger.Pipelines, pipeline)
	}
	return trigger, nil
}

// Add new trigger {Event, Secret, Pipelines}
func (r *RedisStore) Add(trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	log.Debugf("Adding trigger %s ...", trigger.Event)

	// check if there is an existing trigger with assigned pipelines
	count, err := redis.Int(con.Do("SCARD", getTriggerKey(trigger.Event)))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return err
	}
	if count > 0 {
		return model.ErrTriggerAlreadyExists
	}

	// store trigger
	return r.storeSvc.storeTrigger(r, trigger)
}

// Update trigger
func (r *RedisStore) Update(trigger model.Trigger) error {
	log.Debugf("Updating trigger %s ...", trigger.Event)

	// get existing trigger - fail if not found
	_, err := r.Get(trigger.Event)
	if err != nil {
		log.Error(err)
		return err
	}

	// store trigger
	return r.storeSvc.storeTrigger(r, trigger)
}

// Delete trigger by id
func (r *RedisStore) Delete(id string) error {
	con := r.redisPool.GetConn()
	// remove "trigger:" prefix
	id = strings.TrimPrefix(id, "trigger:")
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

// Run trigger pipelines
func (r *RedisStore) Run(id string, vars map[string]string) ([]string, error) {
	var runs []string
	trigger, err := r.Get(id)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	for _, p := range trigger.Pipelines {
		runID, err := r.pipelineSvc.RunPipeline(p.Name, p.RepoOwner, p.RepoName, vars)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if runID != "" {
			runs = append(runs, runID)
		}
	}
	return runs, nil
}

// CheckSecret check trigger secret
func (r *RedisStore) CheckSecret(id string, message string, secret string) error {
	con := r.redisPool.GetConn()
	// remove "trigger:" prefix
	id = strings.TrimPrefix(id, "trigger:")
	log.Debugf("Getting trigger %s ...", id)
	// get secret from String
	triggerSecret, err := redis.String(con.Do("GET", id))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return err
	}
	if triggerSecret != secret {
		// try to check signature
		if util.CheckHmacSignature(message, secret, triggerSecret) {
			return nil
		}
		return errors.New("invalid secret or HMAC signature")
	}
	return nil
}

// Ping Redis and Codefresh services
func (r *RedisStore) Ping() (string, error) {
	con := r.redisPool.GetConn()
	// get pong from Redis
	pong, err := redis.String(con.Do("PING"))
	if err != nil {
		log.Error(err)
		return "Redis Error", err
	}
	// get version from Codefresh API
	err = r.pipelineSvc.Ping()
	if err != nil {
		log.Error(err)
		return "Codefresh Error", err
	}
	return pong, nil
}

// GetPipelines get trigger pipelines by key
func (r *RedisStore) GetPipelines(id string) ([]model.Pipeline, error) {
	con := r.redisPool.GetConn()
	// get pipelines from Set
	pipelines, err := redis.Values(con.Do("SMEMBERS", getTriggerKey(id)))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return nil, err
	} else if err == redis.ErrNil {
		return nil, model.ErrTriggerNotFound
	}

	triggerPipelines := make([]model.Pipeline, 0)
	for _, p := range pipelines {
		var pipeline model.Pipeline
		json.Unmarshal(p.([]byte), &pipeline)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		triggerPipelines = append(triggerPipelines, pipeline)
	}
	return triggerPipelines, nil
}
