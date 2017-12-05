package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/codefresh-io/cf-triggers/pkg/codefresh"
	"github.com/codefresh-io/cf-triggers/pkg/model"
	"github.com/garyburd/redigo/redis"
	log "github.com/sirupsen/logrus"
)

// RedisStore in memory trigger map store
type RedisStore struct {
	pool        *redis.Pool
	pipelineSvc codefresh.PipelineService
}

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

func getKey(id string) string {
	if strings.HasPrefix(id, "trigger:") {
		return id
	}
	return fmt.Sprintf("trigger:%s", id)
}

// NewRedisStore create new Redis DB for storing trigger map
func NewRedisStore(server string, port int, password string, pipelineSvc codefresh.PipelineService) model.TriggerService {
	return &RedisStore{newPool(server, port, password), pipelineSvc}
}

// List get list of defined triggers
func (r *RedisStore) List(filter string) ([]model.Trigger, error) {
	con := r.pool.Get()
	log.Debug("Getting all triggers ...")
	// set * for empty filter
	if filter == "" {
		filter = "*"
	}
	keys, err := redis.Values(con.Do("KEYS", getKey(filter)))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Iterate through all trigger keys and get triggers
	var triggers []model.Trigger
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
	con := r.pool.Get()
	log.Debugf("Getting trigger %s ...", id)
	pipelines, err := redis.Values(con.Do("SMEMBERS", getKey(id)))
	if err != nil {
		log.Error(err)
		return model.Trigger{}, err
	}
	var trigger model.Trigger
	if len(pipelines) > 0 {
		trigger.Event = strings.TrimPrefix(id, "trigger:")
	}
	for _, p := range pipelines {
		var pipeline model.Pipeline
		json.Unmarshal(p.([]byte), &pipeline)
		if err != nil {
			log.Error(err)
			return model.Trigger{}, err
		}
		trigger.Pipelines = append(trigger.Pipelines, pipeline)
	}
	return trigger, nil
}

// Add new trigger
func (r *RedisStore) Add(trigger model.Trigger) error {
	con := r.pool.Get()
	log.Debugf("Adding/Updating trigger %s ...", trigger.Event)
	for _, v := range trigger.Pipelines {
		pipeline, err := json.Marshal(v)
		if err != nil {
			log.Error(err)
			return err
		}
		log.Debugf("trigger '%s' <- '%s' pipeline \n", trigger.Event, pipeline)
		_, err = con.Do("SADD", getKey(trigger.Event), pipeline)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// Delete trigger by id
func (r *RedisStore) Delete(id string) error {
	con := r.pool.Get()
	log.Debugf("Deleting trigger %s ...", id)
	if _, err := con.Do("DEL", getKey(id)); err != nil {
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
	trigger, err := r.Get(id)
	if err != nil {
		log.Error(err)
		return err
	}
	if trigger.Secret != secret {
		return errors.New("invalid secret")
	}
	return nil
}
