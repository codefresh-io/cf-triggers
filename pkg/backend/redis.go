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
	"fmt"
	"strings"
	"time"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
	"github.com/garyburd/redigo/redis"
	log "github.com/sirupsen/logrus"
)

// Redis connection pool
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
	// keep functions as members (for test stubbing)
	getFunc          func(eventURI string) (*model.Trigger, error)
	addFunc          func(trigger model.Trigger) error
	deleteFunc       func(eventURI string) error
	storeTriggerFunc func(trigger model.Trigger) error
}

// helper function - discard Redis transaction and return error
func discardOnError(con redis.Conn, err error) error {
	if _, err := con.Do("DISCARD"); err != nil {
		log.Error(err)
		return err
	}
	log.Error(err)
	return err
}

// construct prefixes - for trigger, secret and pipeline
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
	return getPrefixKey("pipeline", id)
}

// NewRedisStore create new Redis DB for storing trigger map
func NewRedisStore(server string, port int, password string, pipelineSvc codefresh.PipelineService) *RedisStore {
	r := new(RedisStore)
	r.redisPool = &RedisPool{newPool(server, port, password)}
	r.pipelineSvc = pipelineSvc
	// set functions
	r.deleteFunc = r.Delete
	r.storeTriggerFunc = r.StoreTrigger
	r.getFunc = r.Get
	r.addFunc = r.Add
	// return RedisStore
	return r
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
		trigger, err := r.getFunc(eventURI)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

// ListByPipeline get list of defined triggers
func (r *RedisStore) ListByPipeline(pipelineURI string) ([]*model.Trigger, error) {
	con := r.redisPool.GetConn()
	log.Debugf("Getting triggers for pipeline %s..", pipelineURI)
	events, err := redis.Strings(con.Do("ZRANGE", getPipelineKey(pipelineURI), 0, -1))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Iterate through all events and get triggers
	triggers := []*model.Trigger{}
	for _, eventURI := range events {
		// get trigger by eventURI
		trigger, err := r.getFunc(eventURI)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

// Get trigger by eventURI
func (r *RedisStore) Get(eventURI string) (*model.Trigger, error) {
	con := r.redisPool.GetConn()
	log.Debugf("Getting trigger %s ...", eventURI)
	// get secret from String
	secret, err := redis.String(con.Do("GET", getSecretKey(eventURI)))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return nil, err
	}
	// get pipelines from Set
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(eventURI), 0, -1))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return nil, err
	} else if err == redis.ErrNil {
		return nil, model.ErrTriggerNotFound
	}
	trigger := new(model.Trigger)
	if len(pipelines) > 0 {
		trigger.Event = eventURI
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

// StoreTrigger (non-interface) common code for RedisStore.Add() and RedisStore.Update()
func (r *RedisStore) StoreTrigger(trigger model.Trigger) error {
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
		log.Debugf("trigger '%s' <- '%s' pipeline", trigger.Event, pipelineURI)
		// add pipeline to Triggers
		_, err = con.Do("ZADD", getTriggerKey(trigger.Event), 0, pipelineURI)
		if err != nil {
			return discardOnError(con, err)
		}
		// add trigger to Pipelines
		log.Debugf("pipeline '%s' <- '%s' trigger", pipelineURI, trigger.Event)
		_, err = con.Do("ZADD", getPipelineKey(pipelineURI), 0, trigger.Event)
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
	return r.storeTriggerFunc(trigger)
}

// Update trigger
func (r *RedisStore) Update(trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	log.Debugf("Updating trigger %s ...", trigger.Event)

	// check if there is an existing trigger with assigned pipelines
	count, err := redis.Int(con.Do("ZCARD", getTriggerKey(trigger.Event)))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return err
	}
	if count == 0 {
		return model.ErrTriggerNotFound
	}

	// store trigger
	return r.storeTriggerFunc(trigger)
}

// Delete trigger by id
func (r *RedisStore) Delete(eventURI string) error {
	con := r.redisPool.GetConn()
	log.Debugf("Deleting trigger %s ...", eventURI)

	// get pipelines for trigger
	log.Debugf("Get pipelines for eventURI '%s'", eventURI)
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(eventURI), 0, -1))
	if err != nil {
		return discardOnError(con, err)
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		log.Error(err)
		return err
	}

	// delete secret from Secrets (Redis String)
	log.Debugf("Delete secret for eventURI '%s'", eventURI)
	if _, err := con.Do("DEL", getSecretKey(eventURI)); err != nil {
		return discardOnError(con, err)
	}

	// delete eventURI from Pipelines (Redis Sorted Set)
	for _, p := range pipelines {
		log.Debugf("Remove '%s' eventURI from pipeline '%s'", eventURI, p)
		if _, err := con.Do("ZREM", getPipelineKey(p), eventURI); err != nil {
			return discardOnError(con, err)
		}
	}

	// delete trigger from Triggers (Redis Sorted Set)
	log.Debugf("Delete trigger for eventURI '%s'", eventURI)
	if _, err := con.Do("DEL", getTriggerKey(eventURI)); err != nil {
		return discardOnError(con, err)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.Error(err)
	}
	return err
}

// Ping Redis services
func (r *RedisStore) Ping() (string, error) {
	con := r.redisPool.GetConn()
	// get pong from Redis
	pong, err := redis.String(con.Do("PING"))
	if err != nil {
		log.Error(err)
	}
	return pong, err
}

// GetPipelines get trigger pipelines by eventURI
func (r *RedisStore) GetPipelines(eventURI string) ([]model.Pipeline, error) {
	con := r.redisPool.GetConn()
	// get pipelines from Set
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(eventURI), 0, -1))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return nil, err
	} else if err == redis.ErrNil {
		return nil, model.ErrTriggerNotFound
	}
	if len(pipelines) == 0 {
		return nil, model.ErrPipelineNotFound
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

// AddPipelines get trigger pipelines by eventURI
func (r *RedisStore) AddPipelines(eventURI string, pipelines []model.Pipeline) error {
	log.Debugf("Updating trigger %s pipelines ...", eventURI)
	// get existing trigger
	trigger, err := r.getFunc(eventURI)
	if err != nil && err != model.ErrTriggerNotFound {
		log.Error(err)
		return err
	}
	// create trigger if not found
	if err == model.ErrTriggerNotFound {
		trigger = &model.Trigger{}
		trigger.Event = eventURI
	}

	// add pipelines to found/created trigger
	for _, p := range pipelines {
		trigger.Pipelines = append(trigger.Pipelines, p)
	}

	// add trigger if not found
	if err == model.ErrTriggerNotFound {
		return r.addFunc(*trigger)
	}

	// store (update) existing trigger
	return r.storeTriggerFunc(*trigger)
}

// DeletePipeline remove pipeline from trigger
func (r *RedisStore) DeletePipeline(eventURI string, pipelineURI string) error {
	// get Redis connection
	con := r.redisPool.GetConn()
	log.Debugf("Removing pipeline %s from %s ...", pipelineURI, eventURI)

	// try to remove pipeline from set
	result, err := redis.Int(con.Do("ZREM", getTriggerKey(eventURI), pipelineURI))
	if err != nil {
		log.Error(err)
		return err
	}
	// failed to remove
	if result == 0 {
		return model.ErrPipelineNotFound
	}

	// check if there are pipelines left
	count, err := redis.Int(con.Do("ZCARD", getTriggerKey(eventURI)))
	if err != nil && err != redis.ErrNil {
		log.Error(err)
		return err
	}
	// no more pipelines -> delete trigger
	if count == 0 {
		return r.deleteFunc(eventURI)
	}

	return nil
}
