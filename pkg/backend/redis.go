package backend

/*  REDIS Data Model

				Triggers (Sorted Set)

	+-------------------------------------------------+
	|                                                 |
	|                                                 |
	| +---------------------+     +-----------------+ |
	| |                     |     |                 | |
	| | trigger:{event-uri} +-----> {pipeline-uid}  | |
	| |                     |     |                 | |
	| +---------------------+     | ...             | |
	|                             |                 | |
	|                             | {pipeline-uid}  | |
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
	| | pipeline:{pipeline-uid} +------> {event-uri} |  |
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
    * pipeline-uid  - Codefresh pipeline UID

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
				log.WithError(err).Fatal("Failed to connect to Redis store")
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					log.WithError(err).Fatal("Failed to connect to Redis store")
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
		log.WithError(err).Error("Failed to discard Redis transaction")
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

// ListTriggersForEvents get list of defined triggers for trigger events
func (r *RedisStore) ListTriggersForEvents(events []string) ([]model.TriggerLink, error) {
	con := r.redisPool.GetConn()
	log.WithField("events", events).Debug("List triggers for events")

	keys := make([]string, 0)
	for _, event := range events {
		res, err := redis.Strings(con.Do("KEYS", getTriggerKey(event)))
		if err != nil {
			log.WithField("event", event).WithError(err).Error("Failed to find triggers")
			return nil, err
		}
		keys = util.MergeStrings(keys, res)
	}

	// Iterate through all trigger keys and get pipelines
	triggers := make([]model.TriggerLink, 0)
	for _, k := range keys {
		res, err := redis.Strings(con.Do("ZRANGE", k, 0, -1))
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to get pipelines")
			return nil, err
		}
		// for all linked pipelines ...
		for _, pipeline := range res {
			trigger := model.TriggerLink{
				Event:    strings.TrimPrefix(k, "trigger:"),
				Pipeline: pipeline,
			}
			triggers = append(triggers, trigger)
		}
	}
	return triggers, nil
}

// ListTriggersForPipelines get list of defined triggers for specified pipelines
func (r *RedisStore) ListTriggersForPipelines(pipelines []string) ([]model.TriggerLink, error) {
	con := r.redisPool.GetConn()
	log.WithField("pipelines", pipelines).Debug("List triggers for pipelines")

	keys := make([]string, 0)
	for _, pipeline := range pipelines {
		res, err := redis.Strings(con.Do("KEYS", getPipelineKey(pipeline)))
		if err != nil {
			log.WithField("pipeline", pipeline).WithError(err).Error("Failed to find triggers for pipeline")
			return nil, err
		}
		keys = util.MergeStrings(keys, res)
	}

	// Iterate through all pipelines keys and get trigger events
	triggers := make([]model.TriggerLink, 0)
	for _, k := range keys {
		res, err := redis.Strings(con.Do("ZRANGE", k, 0, -1))
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to get trigger events")
			return nil, err
		}
		// for all linked trigger events ...
		for _, event := range res {
			trigger := model.TriggerLink{
				Event:    event,
				Pipeline: strings.TrimPrefix(k, "pipeline:"),
			}
			triggers = append(triggers, trigger)
		}
	}
	return triggers, nil
}

// GetSecret trigger by eventURI
func (r *RedisStore) GetSecret(eventURI string) (string, error) {
	con := r.redisPool.GetConn()
	log.WithField("event-uri", eventURI).Debug("Getting trigger secret")
	// get secret from String
	secret, err := redis.String(con.Do("GET", getSecretKey(eventURI)))
	if err != nil && err != redis.ErrNil {
		log.WithError(err).Error("Failed to get secret")
		return "", err
	}
	return secret, nil
}

// Get trigger by eventURI
func (r *RedisStore) Get(eventURI string) (*model.Trigger, error) {
	con := r.redisPool.GetConn()
	log.WithField("event-uri", eventURI).Debug("Getting trigger")
	// get secret from String
	secret, err := redis.String(con.Do("GET", getSecretKey(eventURI)))
	if err != nil && err != redis.ErrNil {
		log.WithError(err).Error("Failed to get secret")
		return nil, err
	}
	// get pipelines from Set
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(eventURI), 0, -1))
	if err != nil && err != redis.ErrNil {
		log.WithError(err).Error("Failed to get pipelines")
		return nil, err
	} else if err == redis.ErrNil || len(pipelines) == 0 {
		log.Warning("Trigger not found")
		return nil, model.ErrTriggerNotFound
	}
	trigger := new(model.Trigger)
	if len(pipelines) > 0 {
		trigger.Event = eventURI
		trigger.Secret = secret
	}
	trigger.Pipelines = pipelines

	return trigger, nil
}

// CreateTriggersForPipeline create trigger: link multiple events <-> pipeline
func (r *RedisStore) CreateTriggersForPipeline(pipeline string, events []string) error {
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"pipeline-uid": pipeline,
		"event-uri(s)": events,
	}).Debug("Creating triggers")

	// check Codefresh pipeline existence
	_, err := r.pipelineSvc.CheckPipelineExists(pipeline)
	if err != nil {
		return err
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		log.WithError(err).Error("Failed to start Redis transaction")
		return err
	}

	// add pipeline to Triggers
	for _, event := range events {
		log.WithFields(log.Fields{
			"event-uri":    event,
			"pipeline-uid": pipeline,
		}).Debug("Adding pipeline to the Triggers map")
		_, err = con.Do("ZADD", getTriggerKey(event), 0, pipeline)
		if err != nil {
			return discardOnError(con, err)
		}
	}

	// add trigger to Pipelines
	log.WithFields(log.Fields{
		"pipeline-uid": pipeline,
		"event-uri(s)": events,
	}).Debug("Adding trigger to the Pipelines map")
	_, err = con.Do("ZADD", getPipelineKey(pipeline), 0, events)
	if err != nil {
		return discardOnError(con, err)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.WithError(err).Error("Failed to execute transaction")
	}
	return err
}

// DeleteTriggersForPipeline delete trigger: unlink multiple events from pipeline
func (r *RedisStore) DeleteTriggersForPipeline(pipeline string, events []string) error {
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"pipeline-uid": pipeline,
		"event-uri(s)": events,
	}).Debug("Deleting triggers")

	// start Redis transaction
	_, err := con.Do("MULTI")
	if err != nil {
		log.WithError(err).Error("Failed to start Redis transaction")
		return err
	}

	// remove pipeline from Triggers
	for _, event := range events {
		log.WithFields(log.Fields{
			"event-uri":    event,
			"pipeline-uid": pipeline,
		}).Debug("Removing pipeline from the Triggers map")
		_, err = con.Do("ZREM", getTriggerKey(event), pipeline)
		if err != nil {
			return discardOnError(con, err)
		}
	}

	// remove trigger(s) from Pipelines
	log.WithFields(log.Fields{
		"pipeline-uid": pipeline,
		"event-uri(s)": events,
	}).Debug("Removing triggers from the Pipelines map")
	_, err = con.Do("ZREM", getPipelineKey(pipeline), events)
	if err != nil {
		return discardOnError(con, err)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.WithError(err).Error("Failed to execute transaction")
	}
	return err
}

// CreateTriggersForEvent create trigger: link event <-> multiple pipelines
func (r *RedisStore) CreateTriggersForEvent(event string, pipelines []string) error {
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"event-uri":       event,
		"pipeline-uid(s)": pipelines,
	}).Debug("Creating triggers")

	// start Redis transaction
	_, err := con.Do("MULTI")
	if err != nil {
		log.WithError(err).Error("Failed to start Redis transaction")
		return err
	}

	for _, pipeline := range pipelines {
		// check Codefresh pipeline existence
		_, err = r.pipelineSvc.CheckPipelineExists(pipeline)
		if err != nil {
			return discardOnError(con, err)
		}

		// add trigger to Pipelines
		log.WithFields(log.Fields{
			"pipeline-uid": pipeline,
			"event-uri":    event,
		}).Debug("Adding trigger to the Pipelines map")
		_, err = con.Do("ZADD", getPipelineKey(pipeline), 0, event)
		if err != nil {
			return discardOnError(con, err)
		}
	}
	// add pipelines to Triggers
	log.WithFields(log.Fields{
		"event-uri":       event,
		"pipeline-uid(s)": pipelines,
	}).Debug("Adding pipeline to the Triggers map")
	_, err = con.Do("ZADD", getTriggerKey(event), 0, pipelines)
	if err != nil {
		return discardOnError(con, err)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.WithError(err).Error("Failed to execute transaction")
	}
	return err
}

// StoreTrigger (non-interface) common code for RedisStore.Add() and RedisStore.Update()
func (r *RedisStore) StoreTrigger(trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	log.WithField("trigger", trigger).Debug("Storing trigger")
	// generate random secret if required
	if trigger.Secret == model.GenerateKeyword {
		log.Debug("Auto generating trigger secret")
		trigger.Secret = util.RandomString(16)
	}

	// start Redis transaction
	_, err := con.Do("MULTI")
	if err != nil {
		log.WithError(err).Error("Failed to start Redis transaction")
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
	for _, puid := range trigger.Pipelines {
		// check Codefresh pipeline existence
		_, err := r.pipelineSvc.CheckPipelineExists(puid)
		if err != nil {
			return discardOnError(con, err)
		}
		log.WithFields(log.Fields{
			"event-uri":    trigger.Event,
			"pipeline-uid": puid,
		}).Debug("Adding pipeline to trigger map")
		// add pipeline to Triggers
		_, err = con.Do("ZADD", getTriggerKey(trigger.Event), 0, puid)
		if err != nil {
			return discardOnError(con, err)
		}
		// add trigger to Pipelines
		log.WithFields(log.Fields{
			"pipeline-uid": puid,
			"event-uri":    trigger.Event,
		}).Debug("Adding trigger to pipeline map")
		_, err = con.Do("ZADD", getPipelineKey(puid), 0, trigger.Event)
		if err != nil {
			return discardOnError(con, err)
		}
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.WithError(err).Error("Failed to execute transaction")
	}
	return err
}

// Add new trigger {Event, Secret, Pipelines}
func (r *RedisStore) Add(trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	log.WithField("event-uri", trigger.Event).Debug("Adding trigger")

	// check if there is an existing trigger with assigned pipelines
	count, err := redis.Int(con.Do("ZCARD", getTriggerKey(trigger.Event)))
	if err != nil && err != redis.ErrNil {
		log.WithError(err).Error("Failed to get number of pipelines")
		return err
	}
	if count > 0 {
		log.Debug("Trigger already exists")
		return model.ErrTriggerAlreadyExists
	}

	// store trigger
	return r.storeTriggerFunc(trigger)
}

// Update trigger
func (r *RedisStore) Update(trigger model.Trigger) error {
	con := r.redisPool.GetConn()
	log.WithField("event-uri", trigger.Event).Debug("Updating trigger")

	// check if there is an existing trigger with assigned pipelines
	count, err := redis.Int(con.Do("ZCARD", getTriggerKey(trigger.Event)))
	if err != nil && err != redis.ErrNil {
		log.WithError(err).Error("Failed to get number of pipelines")
		return err
	}
	if count == 0 {
		log.Error("Trigger not found")
		return model.ErrTriggerNotFound
	}

	// store trigger
	return r.storeTriggerFunc(trigger)
}

// Delete trigger by id
func (r *RedisStore) Delete(eventURI string) error {
	con := r.redisPool.GetConn()
	log.WithField("event-uri", eventURI).Debug("Deleting trigger")

	// get pipelines for trigger
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(eventURI), 0, -1))
	if err != nil {
		log.WithError(err).Debug("Failed to get pipelines")
		return discardOnError(con, err)
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		log.WithError(err).Error("Failed to start Redis transaction")
		return err
	}

	// delete secret from Secrets (Redis String)
	log.WithField("event-uri", eventURI).Debug("Delete secret")
	if _, err := con.Do("DEL", getSecretKey(eventURI)); err != nil {
		return discardOnError(con, err)
	}

	// delete eventURI from Pipelines (Redis Sorted Set)
	for _, puid := range pipelines {
		log.WithFields(log.Fields{
			"event-uri":    eventURI,
			"pipeline-uid": puid,
		}).Debug("Remove trigger from pipeline map")
		if _, err := con.Do("ZREM", getPipelineKey(puid), eventURI); err != nil {
			return discardOnError(con, err)
		}
	}

	// delete trigger from Triggers (Redis Sorted Set)
	log.WithField("event-uri", eventURI).Debug("Delete trigger")
	if _, err := con.Do("DEL", getTriggerKey(eventURI)); err != nil {
		return discardOnError(con, err)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.WithError(err).Error("Failed to execute Redis transaction")
	}
	return err
}

// Ping Redis services
func (r *RedisStore) Ping() (string, error) {
	con := r.redisPool.GetConn()
	// get pong from Redis
	pong, err := redis.String(con.Do("PING"))
	if err != nil {
		log.WithError(err).Error("Failed to ping Redis server")
	}
	return pong, err
}

// GetPipelinesForTriggers get pipelines that have trigger defined
// can be filtered by event-uri(s)
func (r *RedisStore) GetPipelinesForTriggers(events []string) ([]string, error) {
	con := r.redisPool.GetConn()

	// accumulator array
	var all []string

	// using events filter
	if len(events) > 0 {
		for _, event := range events {
			log.WithField("event-uri", event).Debug("Getting pipelines for trigger filter")
			// get pipelines from Trigger Set
			pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(event), 0, -1))
			if err != nil && err != redis.ErrNil {
				log.WithError(err).Error("Failed to get pipelines")
				return nil, err
			} else if err == redis.ErrNil {
				log.Warning("No pipelines found")
				return nil, model.ErrTriggerNotFound
			}
			if len(pipelines) == 0 {
				log.Warning("No pipelines found")
				return nil, model.ErrPipelineNotFound
			}
			// aggregate pipelines
			all = util.MergeStrings(all, pipelines)
		}
	} else { // getting all pipelines
		log.Debug("Getting all pipelines")
		// get pipelines from Trigger Set
		pipelines, err := redis.Strings(con.Do("KEYS", getPipelineKey("")))
		if err != nil && err != redis.ErrNil {
			log.WithError(err).Error("Failed to get pipelines")
			return nil, err
		} else if err == redis.ErrNil {
			log.Warning("No pipelines found")
			return nil, model.ErrTriggerNotFound
		}
		if len(pipelines) == 0 {
			log.Warning("No pipelines found")
			return nil, model.ErrPipelineNotFound
		}
		all = append(all, pipelines...)
	}

	return all, nil
}
