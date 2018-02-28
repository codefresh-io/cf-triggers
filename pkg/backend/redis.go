package backend

/*  REDIS Data Model

				Trigger-Events (Hash)

    +---------------------------------------------+
    |                                             |
    |                                             |
    | +--------------------+      +-------------+ |
    | |                    |      |             | |
    | | event:{event-uri}  +------> type        | |
    | |                    |      | kind        | |
    | |                    |      | secret      | |
	| |                    |      | endpoint    | |
	| |                    |      | account*    | |
    | |                    |      | description | |
    | |                    |      | help        | |
    | |                    |      | status      | |
    | |                    |      |             | |
    | +--------------------+      +-------------+ |
    |                                             |
    |                                             |
	+---------------------------------------------+

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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/provider"
	"github.com/codefresh-io/hermes/pkg/util"
	"github.com/garyburd/redigo/redis"
	log "github.com/sirupsen/logrus"
)

func getAccount(ctx context.Context) string {
	v := ctx.Value(model.ContextKeyAccount)
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

func getPublicFlag(ctx context.Context) bool {
	v := ctx.Value(model.ContextKeyPublic)
	if flag, ok := v.(bool); ok {
		return flag
	}
	return false
}

// Redis connection pool
func newPool(server string, port int, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", server, port))
			if err != nil {
				log.WithError(err).Fatal("failed to connect to the Redis store")
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					log.WithError(err).Fatal("failed to connect to the Redis store")
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
	redisPool     RedisPoolService
	pipelineSvc   codefresh.PipelineService
	eventProvider provider.EventProvider
}

// helper function - discard Redis transaction and return error
func discardOnError(con redis.Conn, err error) error {
	if _, err := con.Do("DISCARD"); err != nil {
		log.WithError(err).Error("failed to discard Redis transaction")
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

func getAccountSuffixKey(account, id string) string {
	suffix := model.CalculateAccountHash(account)
	// if id already has private ot public account suffix, return it as is
	// also skip adding suffix to id for "-" account
	if account == "-" || strings.HasSuffix(id, suffix) || strings.HasSuffix(id, model.PublicAccountHash) {
		return id
	}
	return fmt.Sprintf("%s:%s", id, suffix)
}

func getTriggerKey(account, id string) string {
	key := getPrefixKey("trigger", id)
	return getAccountSuffixKey(account, key)
}

// pipeline key is not account aware
func getPipelineKey(id string) string {
	return getPrefixKey("pipeline", id)
}

func getEventKey(account, id string) string {
	key := getPrefixKey("event", id)
	return getAccountSuffixKey(account, key)
}

// NewRedisStore create new Redis DB for storing trigger map
func NewRedisStore(server string, port int, password string, pipelineSvc codefresh.PipelineService, eventProvider provider.EventProvider) *RedisStore {
	r := new(RedisStore)
	r.redisPool = &RedisPool{newPool(server, port, password)}
	r.pipelineSvc = pipelineSvc
	r.eventProvider = eventProvider
	// return RedisStore
	return r
}

//-------------------------- TriggerReaderWriter Interface -------------------------

// GetEventTriggers get list of triggers for specified event
func (r *RedisStore) GetEventTriggers(ctx context.Context, event string) ([]model.Trigger, error) {
	account := getAccount(ctx)
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"event":   event,
		"account": account,
	}).Debug("get triggers for event")

	// get trigger keys for events
	keys, err := redis.Strings(con.Do("KEYS", getTriggerKey(account, event)))
	if err != nil {
		log.WithField("event", event).WithError(err).Error("failed to find triggers")
		return nil, err
	}
	// get trigger keys for matching public events
	publicKeys, err := redis.Strings(con.Do("KEYS", getTriggerKey(model.PublicAccount, event)))
	if err != nil {
		log.WithField("event", event).WithError(err).Error("failed to find triggers")
		return nil, err
	}
	// merge keys
	keys = util.MergeStrings(publicKeys, keys)

	// Iterate through all trigger keys and get linked pipelines
	triggers := make([]model.Trigger, 0)
	for _, k := range keys {
		res, err := redis.Strings(con.Do("ZRANGE", k, 0, -1))
		if err != nil {
			log.WithField("key", k).WithError(err).Error("failed to get pipelines")
			return nil, err
		}
		// for all linked pipelines ...
		for _, pipeline := range res {
			trigger := model.Trigger{
				Event:    strings.TrimPrefix(k, "trigger:"),
				Pipeline: pipeline,
			}
			triggers = append(triggers, trigger)
		}
	}
	if len(triggers) == 0 {
		return nil, model.ErrTriggerNotFound
	}
	return triggers, nil
}

// GetPipelineTriggers get list of defined triggers for specified pipeline
func (r *RedisStore) GetPipelineTriggers(ctx context.Context, pipeline string) ([]model.Trigger, error) {
	account := getAccount(ctx)
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"pipeline": pipeline,
		"account":  account,
	}).Debug("get triggers for pipeline")

	pipelineKey := getPipelineKey(pipeline)
	n, err := redis.Int(con.Do("EXISTS", pipelineKey))
	if err != nil || n == 0 {
		log.WithField("pipeline", pipeline).WithError(err).Error("failed to find triggers for pipeline")
		return nil, err
	}

	// Iterate through all pipelines keys and get trigger events (public) and per account
	suffix := model.CalculateAccountHash(account)
	triggers := make([]model.Trigger, 0)

	res, err := redis.Strings(con.Do("ZRANGE", pipelineKey, 0, -1))
	if err != nil {
		log.WithField("key", pipelineKey).WithError(err).Error("failed to get trigger events")
		return nil, err
	}
	// for all linked trigger events, check if event belongs to context account of it's a public event
	for _, event := range res {
		if strings.HasSuffix(event, suffix) || strings.HasSuffix(event, model.PublicAccountHash) {
			trigger := model.Trigger{
				Event:    event,
				Pipeline: pipeline,
			}
			triggers = append(triggers, trigger)
		}
	}
	if len(triggers) == 0 {
		return nil, model.ErrTriggerNotFound
	}
	return triggers, nil
}

// DeleteTrigger delete trigger: unlink event from pipeline
func (r *RedisStore) DeleteTrigger(ctx context.Context, event, pipeline string) error {
	account := getAccount(ctx)
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"pipeline": pipeline,
		"event":    event,
		"account":  account,
	}).Debug("deleting trigger")

	// check event account match: public or private
	if !model.MatchPublicAccount(event) && !model.MatchAccount(account, event) {
		return model.ErrTriggerNotFound
	}

	// check Codefresh pipeline match; ignore all errors beside "no match"
	_, err := r.pipelineSvc.GetPipeline(account, pipeline)
	if err == codefresh.ErrPipelineNoMatch {
		log.WithError(err).Error("attempt to remove pipeline from another account")
		return err
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		log.WithError(err).Error("failed to start Redis transaction")
		return err
	}

	// remove pipeline from Triggers
	_, err = con.Do("ZREM", getTriggerKey(account, event), pipeline)
	if err != nil {
		return discardOnError(con, err)
	}

	// remove trigger(s) from Pipelines
	_, err = con.Do("ZREM", getPipelineKey(pipeline), event)
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

// CreateTrigger create trigger: link event <-> multiple pipelines
func (r *RedisStore) CreateTrigger(ctx context.Context, event, pipeline string) error {
	account := getAccount(ctx)
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"event":    event,
		"pipeline": pipeline,
		"account":  account,
	}).Debug("Creating triggers")

	// check event account match: public or private
	if !model.MatchPublicAccount(event) && !model.MatchAccount(account, event) {
		return model.ErrTriggerNotFound
	}

	// check Codefresh pipeline existence
	_, err := r.pipelineSvc.GetPipeline(account, pipeline)
	if err != nil {
		return err
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		log.WithError(err).Error("failed to start Redis transaction")
		return err
	}

	// add trigger to Pipelines
	_, err = con.Do("ZADD", getPipelineKey(pipeline), 0, event)
	if err != nil {
		return discardOnError(con, err)
	}

	// add pipeline to Triggers
	_, err = con.Do("ZADD", getTriggerKey(account, event), pipeline)
	if err != nil {
		return discardOnError(con, err)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		log.WithError(err).Error("failed to execute transaction")
	}
	return err
}

// GetTriggerPipelines get pipelines that have trigger defined
// can be filtered by event-uri(s)
func (r *RedisStore) GetTriggerPipelines(ctx context.Context, event string) ([]string, error) {
	account := getAccount(ctx)
	con := r.redisPool.GetConn()

	log.WithFields(log.Fields{
		"event":   event,
		"account": account,
	}).Debug("getting pipelines for trigger event")
	// get pipelines from Triggers
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(account, event), 0, -1))
	if err != nil && err != redis.ErrNil {
		log.WithError(err).Error("failed to get pipelines")
		return nil, err
	} else if err == redis.ErrNil {
		return nil, model.ErrTriggerNotFound
	}
	if len(pipelines) == 0 {
		return nil, model.ErrPipelineNotFound
	}

	return pipelines, nil
}

// CreateEvent new trigger event
func (r *RedisStore) CreateEvent(ctx context.Context, eventType, kind, secret, context string, values map[string]string) (*model.Event, error) {
	account := getAccount(ctx)
	// replace account to public account for public event creation
	public := getPublicFlag(ctx)
	if public {
		account = model.PublicAccount
	}

	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"type":    eventType,
		"kind":    kind,
		"values":  values,
		"account": account,
	}).Debug("Creating a new trigger event")

	// construct event URI
	eventURI, err := r.eventProvider.ConstructEventURI(eventType, kind, account, values)
	if err != nil {
		return nil, err
	}

	// generate random secret if required
	if secret == model.GenerateKeyword {
		log.Debug("Auto generating trigger secret")
		secret = util.RandomString(16)
	}

	// TODO: get credentials from Codefresh context
	var credentials map[string]string

	// try subscribing to event - create event in remote system through event provider
	eventInfo, err := r.eventProvider.SubscribeToEvent(eventURI, secret, credentials)
	if err != nil {
		if err == provider.ErrNotImplemented {
			// try to get event info (required method)
			eventInfo, err = r.eventProvider.GetEventInfo(eventURI, secret)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	event := model.Event{
		URI:       eventURI,
		Type:      eventType,
		Kind:      kind,
		Account:   account,
		Secret:    secret,
		EventInfo: *eventInfo,
	}

	// start Redis transaction
	eventKey := getEventKey(account, eventURI)
	if _, err := con.Do("MULTI"); err != nil {
		log.WithError(err).Error("Failed to start Redis transaction")
		return nil, err
	}
	// store event type
	if _, err := con.Do("HSETNX", eventKey, "type", eventType); err != nil {
		return nil, discardOnError(con, err)
	}
	// store event kind
	if _, err := con.Do("HSETNX", eventKey, "kind", kind); err != nil {
		return nil, discardOnError(con, err)
	}
	// store event account, if not empty
	if _, err := con.Do("HSETNX", eventKey, "account", account); err != nil {
		return nil, discardOnError(con, err)
	}
	// store event secret
	if _, err := con.Do("HSETNX", eventKey, "secret", secret); err != nil {
		return nil, discardOnError(con, err)
	}
	// store event description (from event provider)
	if _, err := con.Do("HSETNX", eventKey, "description", eventInfo.Description); err != nil {
		return nil, discardOnError(con, err)
	}
	// store event endpoint (from event provider)
	if _, err := con.Do("HSETNX", eventKey, "endpoint", eventInfo.Endpoint); err != nil {
		return nil, discardOnError(con, err)
	}
	// store event help (from event provider)
	if _, err := con.Do("HSETNX", eventKey, "help", eventInfo.Help); err != nil {
		return nil, discardOnError(con, err)
	}
	// store event status (from event provider)
	if _, err := con.Do("HSETNX", eventKey, "status", eventInfo.Status); err != nil {
		return nil, discardOnError(con, err)
	}
	// submit transaction
	if _, err := con.Do("EXEC"); err != nil {
		log.WithError(err).Error("failed to execute transaction")
		return nil, err
	}
	return &event, nil
}

// GetEvent get event by event URI
func (r *RedisStore) GetEvent(ctx context.Context, event string) (*model.Event, error) {
	account := getAccount(ctx)
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"event-uri": event,
		"account":   account,
	}).Debug("Getting trigger event")
	// prepare key
	eventKey := getEventKey(account, event)
	// check event URI is a single event key
	n, err := redis.Int(con.Do("EXISTS", eventKey))
	if err != nil {
		log.WithError(err).Error("failed to check trigger event existence")
		return nil, err
	}
	if n != 1 {
		log.Error("trigger event key does not exist")
		return nil, model.ErrEventNotFound
	}

	// get hash values
	fields, err := redis.StringMap(con.Do("HGETALL", eventKey))
	if err != nil {
		log.WithError(err).Error("Failed to get trigger event fields")
		return nil, err
	}
	if len(fields) == 0 {
		log.Error("Failed to find trigger event")
		return nil, model.ErrEventNotFound
	}
	return model.StringsMapToEvent(event, fields), nil
}

// GetEvents get events by event type, kind and filter (can be URI or part of URI)
func (r *RedisStore) GetEvents(ctx context.Context, eventType, kind, filter string) ([]model.Event, error) {
	account := getAccount(ctx)
	public := getPublicFlag(ctx)
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"type":    eventType,
		"kind":    kind,
		"account": account,
		"filter":  filter,
	}).Debug("Getting trigger events")
	// get all events URIs for account
	uris, err := redis.Strings(con.Do("KEYS", getEventKey(account, filter)))
	if err != nil {
		log.WithError(err).Error("failed to get trigger events")
		return nil, err
	}
	// get public trigger events, if asked (through context)
	if public {
		// get all events URIs for account
		publicURIs, err := redis.Strings(con.Do("KEYS", getEventKey(model.PublicAccount, filter)))
		if err != nil && err != redis.ErrNil {
			log.WithError(err).Error("failed to get public trigger events")
			return nil, err
		}
		if len(publicURIs) > 0 {
			uris = append(uris, publicURIs...)
		}
	}
	// scan through all events and select matching to non-empty type and kind
	events := make([]model.Event, 0)
	for _, uri := range uris {
		event, err := r.GetEvent(ctx, uri)
		if err != nil {
			return nil, err
		}
		if (eventType == "" || event.Type == eventType) &&
			(kind == "" || event.Kind == kind) {
			events = append(events, *event)
		}
	}
	return events, nil
}

// DeleteEvent delete trigger event
func (r *RedisStore) DeleteEvent(ctx context.Context, event, context string) error {
	account := getAccount(ctx)
	con := r.redisPool.GetConn()
	log.WithFields(log.Fields{
		"event-uri": event,
		"account":   account,
	}).Debug("deleting trigger event")
	// prepare keys
	eventKey := getEventKey(account, event)
	triggerKey := getTriggerKey(account, event)
	// check event URI is a single event key
	n, err := redis.Int(con.Do("EXISTS", eventKey))
	if err != nil {
		log.WithError(err).Error("failed to check trigger event existence")
		return err
	}
	if n == 0 {
		log.Error("trigger event key does not exist")
		return model.ErrEventNotFound
	}
	// check trigger event account vs passed account; skip 'public' events
	a, err := redis.String(con.Do("HGET", eventKey, "account"))
	if err != nil && err != redis.ErrNil {
		log.WithError(err).Error("failed to get trigger event account")
		return err
	}
	// if not public and belongs to different account - return not exists error
	if a != model.PublicAccount && a != account {
		log.Error("trigger event account does not match")
		return model.ErrEventNotFound
	}

	// TODO: get credentials from Codefresh context
	// credentials := make(map[string]string)

	// get pipelines linked to the trigger event
	pipelines, err := redis.Strings(con.Do("ZRANGE", triggerKey, 0, -1))
	if err != nil {
		log.WithError(err).Error("Failed to get pipelines for the trigger event")
		return err
	}

	// abort delete operation if trigger event has linked pipelines
	if len(pipelines) > 0 {
		return model.ErrEventDeleteWithTriggers
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		log.WithError(err).Error("Failed to start Redis transaction")
		return err
	}

	// delete event hash for key
	log.Debug("Removing trigger event")
	_, err = con.Do("DEL", eventKey)
	if err != nil {
		log.WithError(err).Error("Failed to delete trigger event")
		return discardOnError(con, err)
	}

	// delete trigger event from Triggers
	log.Debug("Removing trigger event from Triggers")
	_, err = con.Do("DEL", triggerKey)
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

//-------------------------- Pinger Interface -------------------------

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
