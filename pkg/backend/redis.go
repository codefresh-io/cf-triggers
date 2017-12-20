package backend

/*  REDIS Data Model

	Secrets (String)

	+------------------------------------------+
	|                                          |
	|                                          |
	| +--------------------+      +----------+ |
	| |                    |      |          | |
	| | secret:{event-uri} +------> {secret} | |
	| |                    |      |          | |
	| +--------------------+      +----------+ |
	|                                          |
	|                                          |
	+------------------------------------------+


				Triggers (Sorted Set)

	+-------------------------------------------------+
	|                                                 |
	|                                                 |
	| +---------------------+     +-----------------+ |
	| |                     |     |                 | |
	| | trigger:{event-uri} +-----> {pipeline-uri}  | |
	| |                     |     |                 | |
	| +---------------------+     | ...             | |
	|                             |                 | |
	|                             | {pipeline-uri}  | |
	|                             |                 | |
	|                             +-----------------+ |
	|                                                 |
	|                                                 |
	+-------------------------------------------------+


				Pipelines (Sorted Set)

	+---------------------------------------------------+
	|                                                   |
	|                                                   |
	| +-------------------------+      +-------------+  |
	| |                         |      |             |  |
	| | pipeline:{pipeline-uri} +------> {event-uri} |  |
	| |                         |      |             |  |
	| +-------------------------+      | ...         |  |
	|                                  |             |  |
	|                                  | {event-uri} |  |
	|                                  |             |  |
	|                                  +-------------+  |
	|                                                   |
	|                                                   |
	+---------------------------------------------------+

	* event-uri     - URI unique identifier for event (specified by event provider)
    * pipeline-uri  - Codefresh pipeline URI {repo-owner}:{repo-name}:{name}

*/

import (
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

// helper - discard Redis transaction and return error
func discardOnError(con redis.Conn, err error) error {
	if _, err := con.Do("DISCARD"); err != nil {
		log.Error(err)
		return err
	}
	log.Error(err)
	return err
}

// common code for RedisStore.Add() and RedisStore.Update()
func (redisStoreInternal) storeTrigger(r *RedisStore, trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	// generate random secret if required
	if trigger.Secret == model.GenerateKeyword {
		trigger.Secret = util.RandomString(16)
	}

	// start Redis transaction
	_, err := con.Do("MULTI")
	if err != nil {
		log.Error(err)
		return err
	}

	// add secret to Secrets (Redis STRING)
	if trigger.Secret != "" {
		_, err := con.Do("SET", getSecretKey(trigger.Event), trigger.Secret)
		if err != nil {
			return discardOnError(con, err)
		}
	}

	// add pipelines to Triggers (Redis Sorted Set) and trigger to Pipelines (Sorted Set)
	for _, v := range trigger.Pipelines {
		// check Codefresh pipeline existence
		err := r.pipelineSvc.CheckPipelineExist(v.RepoOwner, v.RepoName, v.Name)
		if err != nil {
			return discardOnError(con, err)
		}
		// create pipeline URI
		pipelineURI := model.PipelineToURI(&v)
		log.Debugf("trigger '%s' <- '%s' pipeline \n", trigger.Event, pipelineURI)
		// add pipeline to Triggers
		_, err = con.Do("ZADD", getTriggerKey(trigger.Event), pipelineURI)
		if err != nil {
			return discardOnError(con, err)
		}
		// add trigger to Pipelines
		log.Debugf("pipeline '%s' <- '%s' trigger \n", pipelineURI, trigger.Event)
		_, err = con.Do("ZADD", getPipelineKey(pipelineURI), trigger.Event)
		if err != nil {
			return discardOnError(con, err)
		}
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.Error(err)
	}
	return err
}

func getPrefixKey(prefix, id string) string {
	// set * for empty id
	if id == "" {
		id = "*"
	}
	if strings.HasPrefix(id, prefix+":") {
		return id
	}
	return fmt.Sprintf("%s:%s", prefix, id)
}

func getTriggerKey(id string) string {
	return getPrefixKey("trigger", id)
}

func getSecretKey(id string) string {
	return getPrefixKey("secret", id)
}

func getPipelineKey(id string) string {
	return getPrefixKey("secret", id)
}

// NewRedisStore create new Redis DB for storing trigger map
func NewRedisStore(server string, port int, password string, pipelineSvc codefresh.PipelineService) model.TriggerService {
	return &RedisStore{&RedisPool{newPool(server, port, password)}, pipelineSvc, &redisStoreInternal{}}
}

// List get list of defined triggers
func (r *RedisStore) List(filter string) ([]*model.Trigger, error) {
	con := r.redisPool.GetConn()
	log.Debug("Getting all triggers ...")
	keys, err := redis.Strings(con.Do("KEYS", getTriggerKey(filter)))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Iterate through all trigger keys and get triggers
	triggers := []*model.Trigger{}
	for _, k := range keys {
		// trim trigger: prefix before Get()
		eventURI := strings.TrimPrefix(k, "trigger:")
		// get trigger by eventURI
		trigger, err := r.Get(eventURI)
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
	log.Debugf("Getting trigger %s ...", id)
	// get secret from String
	secret, err := redis.String(con.Do("GET", getSecretKey(id)))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return nil, err
	}
	// get pipelines from Set
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(id), 0, -1))
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
		pipeline, err := model.PipelineFromURI(p)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		trigger.Pipelines = append(trigger.Pipelines, *pipeline)
	}
	return trigger, nil
}

// Add new trigger {Event, Secret, Pipelines}
func (r *RedisStore) Add(trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	log.Debugf("Adding trigger %s ...", trigger.Event)

	// check if there is an existing trigger with assigned pipelines
	count, err := redis.Int(con.Do("ZCARD", getTriggerKey(trigger.Event)))
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
	log.Debugf("Deleting trigger %s ...", id)
	// start Redis transaction
	_, err := con.Do("MULTI")
	if err != nil {
		log.Error(err)
		return err
	}

	// delete secret from Secrets (Redis String)
	if _, err := con.Do("DEL", getSecretKey(id)); err != nil {
		return discardOnError(con, err)
	}
	// get pipelines for trigger
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(id), 0, -1))
	if err != nil {
		return discardOnError(con, err)
	}
	// delete eventURI from Pipelines (Redis Sorted Set)
	for _, p := range pipelines {
		if _, err := con.Do("ZREM", getPipelineKey(p), id); err != nil {
			return discardOnError(con, err)
		}
	}

	// delete trigger from Triggers (Redis Sorted Set)
	if _, err := con.Do("DEL", getTriggerKey(id)); err != nil {
		return discardOnError(con, err)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.Error(err)
	}
	return err
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
		runID, err := r.pipelineSvc.RunPipeline(p.RepoOwner, p.RepoName, p.Name, vars)
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
	log.Debugf("Getting trigger %s ...", id)
	// get secret from String
	triggerSecret, err := redis.String(con.Do("GET", getSecretKey(id)))
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
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(id), 0, -1))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return nil, err
	} else if err == redis.ErrNil {
		return nil, model.ErrTriggerNotFound
	}
	if len(pipelines) == 0 {
		return nil, model.ErrTriggerNotFound
	}

	triggerPipelines := make([]model.Pipeline, 0)
	for _, p := range pipelines {
		pipeline, err := model.PipelineFromURI(p)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		triggerPipelines = append(triggerPipelines, *pipeline)
	}
	return triggerPipelines, nil
}

// AddPipelines get trigger pipelines by key
func (r *RedisStore) AddPipelines(id string, pipelines []model.Pipeline) error {
	log.Debugf("Updating trigger %s pipelines ...", id)
	// get existing trigger - fail if not found
	trigger, err := r.Get(id)
	if err != nil {
		log.Error(err)
		return err
	}

	// add pipelines to found trigger
	for _, p := range pipelines {
		trigger.Pipelines = append(trigger.Pipelines, p)
	}

	// store trigger
	return r.storeSvc.storeTrigger(r, *trigger)
}

// DeletePipeline remove pipeline from trigger
func (r *RedisStore) DeletePipeline(id string, pid string) error {
	// get Redis connection
	con := r.redisPool.GetConn()
	log.Debugf("Removing pipeline %s from %s ...", pid, id)
	// check if trigger exists == secret with eventURI exists
	_, err := redis.String(con.Do("GET", getSecretKey(id)))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return err
	}
	// remove pipeline from set
	result, err := redis.Int(con.Do("ZREM", getTriggerKey(id), pid))
	if err != nil {
		log.Error(err)
		return err
	}
	// failed to remove
	if result == 0 {
		return model.ErrPipelineNotFound
	}
	return nil
}
