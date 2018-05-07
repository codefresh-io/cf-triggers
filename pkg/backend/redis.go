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

				Filters (Hash)

    +------------------------------------------------------------+
    |                                                            |
    | +-----------------------------------+      +-------------+ |
    | |                                   |      |             | |
    | | filter:{event-uri}-{pipeline-uid} +------> filter+     | |
    | |                                   |      |             | |
    | +-----------------------------------+      +-------------+ |
    |                                                            |
	+------------------------------------------------------------+

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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/codefresh-io/go-infra/pkg/logger"
	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/provider"
	"github.com/codefresh-io/hermes/pkg/util"
	"github.com/garyburd/redigo/redis"
	"github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
)

func getAccount(ctx context.Context) string {
	v := ctx.Value(model.ContextKeyAccount)
	if str, ok := v.(string); ok {
		return str
	}
	return model.PublicAccount
}

func getNewRelicTransaction(c context.Context) newrelic.Transaction {
	if v := c.Value(model.ContextNewRelicTxn); v != nil {
		if txn, ok := v.(newrelic.Transaction); ok {
			return txn
		}
	}
	return nil
}

// construct Logrus fields from context (requestID and auth context)
func getContextLogFields(ctx context.Context) log.Fields {
	fields := make(log.Fields)
	// get correlation ID
	if requestID, ok := ctx.Value(model.ContextRequestID).(string); ok {
		fields[logger.FieldCorrelationID] = requestID
	}
	// get NewRelic transaction
	if txn, ok := ctx.Value(model.ContextNewRelicTxn).(newrelic.Transaction); ok {
		fields[logger.FieldNewRelicTxn] = txn
	}
	// get auth entity
	if authEntity, ok := ctx.Value(model.ContextAuthEntity).(string); ok {
		data, err := base64.StdEncoding.DecodeString(authEntity)
		if err != nil {
			log.WithError(err).Error("failed to decode authenticated entity")
			return fields
		}
		// auth entity can be user {_id, name} or service {serviceName, type}
		type _auth struct {
			// user fields
			Name string `json:"name,omitempty"`
			ID   string `json:"_id,omitempty"`
			// service fields
			ServiceName string `json:"serviceName,omitempty"`
			Type        string `json:"type,omitempty"`
		}
		auth := new(_auth)
		err = json.Unmarshal(data, auth)
		if err != nil {
			log.WithError(err).Error("failed to load authenticated entity JSON")
			return fields

		}
		// set type to user if empty
		if auth.Type == "" {
			auth.Type = "user"
		}
		// set id to none if empty
		if auth.ID == "" {
			auth.ID = "none"
		}
		// set name to serviceName
		if auth.Name == "" {
			auth.Name = auth.ServiceName
		}
		// set log fields
		fields[logger.FieldAuthName] = auth.Name
		fields[logger.FieldAuthID] = auth.ID
		fields[logger.FieldAuthType] = auth.Type
	}
	return fields
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

// RedisEventGetter implements GetEvent used internally in RedisStore
type RedisEventGetter struct {
	redisPool RedisPoolService
}

// RedisStore in memory trigger map store
type RedisStore struct {
	redisPool     RedisPoolService
	pipelineSvc   codefresh.PipelineService
	eventProvider provider.EventProvider
	eventGetter   model.TriggerEventGetter
}

// helper function - discard Redis transaction and return error
func discardOnError(con redis.Conn, err error, log *log.Entry) error {
	if _, e := con.Do("DISCARD"); e != nil {
		log.WithError(e).Error("failed to discard Redis transaction")
		return e
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

// filter key is not account aware
func getFilterKey(event, pipeline string) string {
	return getPrefixKey("filter", fmt.Sprintf("%s-%s", event, pipeline))
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
	// create
	r.eventGetter = &RedisEventGetter{r.redisPool}
	// return RedisStore
	return r
}

//-------------------------- TriggerReaderWriter Interface -------------------------

// GetEventTriggers get list of triggers for specified event
func (r *RedisStore) GetEventTriggers(ctx context.Context, event string) ([]model.Trigger, error) {
	account := getAccount(ctx)
	lg := log.WithFields(getContextLogFields(ctx))
	lg.WithFields(log.Fields{
		"event":   event,
		"account": account,
	}).Debug("get triggers for event")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()

	// get trigger keys for events
	keys, err := redis.Strings(con.Do("KEYS", getTriggerKey(account, event)))
	if err != nil {
		lg.WithField("event", event).WithError(err).Error("failed to find triggers")
		return nil, err
	}
	// get trigger keys for matching public events
	publicKeys, err := redis.Strings(con.Do("KEYS", getTriggerKey(model.PublicAccount, event)))
	if err != nil {
		lg.WithField("event", event).WithError(err).Error("failed to find triggers")
		return nil, err
	}
	// merge keys
	keys = util.MergeStrings(publicKeys, keys)

	// Iterate through all trigger keys and get linked pipelines
	triggers := make([]model.Trigger, 0)
	for _, k := range keys {
		res, err := redis.Strings(con.Do("ZRANGE", k, 0, -1))
		if err != nil {
			lg.WithField("key", k).WithError(err).Error("failed to get pipelines")
			return nil, err
		}
		// for all linked pipelines ...
		uri := strings.TrimPrefix(k, "trigger:")
		for _, pipeline := range res {
			// get filters
			filters, err := redis.StringMap(con.Do("HGETALL", getFilterKey(uri, pipeline)))
			if err != nil && err != redis.ErrNil {
				lg.WithError(err).Error("error getting trigger filter")
				return nil, err
			}
			// populate trigger object
			trigger := model.Trigger{
				Event:    uri,
				Pipeline: pipeline,
				Filters:  filters,
			}
			triggers = append(triggers, trigger)
		}
	}
	return triggers, nil
}

// GetPipelineTriggers get list of defined triggers for specified pipeline
func (r *RedisStore) GetPipelineTriggers(ctx context.Context, pipeline string, withEvent bool) ([]model.Trigger, error) {
	account := getAccount(ctx)
	lg := log.WithFields(getContextLogFields(ctx))
	lg.WithFields(log.Fields{
		"pipeline": pipeline,
		"account":  account,
	}).Debug("get triggers for pipeline")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()
	pipelineKey := getPipelineKey(pipeline)
	n, err := redis.Int(con.Do("EXISTS", pipelineKey))
	if err != nil {
		lg.WithField("pipeline", pipeline).WithError(err).Error("error finding triggers for pipeline")
		return nil, err
	}
	if n == 0 {
		lg.WithField("pipeline", pipeline).Warn("failed to find triggers for pipeline")
		return nil, nil
	}

	// Iterate through all pipelines keys and get trigger events (public) and per account
	suffix := model.CalculateAccountHash(account)
	triggers := make([]model.Trigger, 0)

	res, err := redis.Strings(con.Do("ZRANGE", pipelineKey, 0, -1))
	if err != nil {
		lg.WithField("key", pipelineKey).WithError(err).Error("failed to get trigger events")
		return nil, err
	}
	// for all linked trigger events, check if event belongs to context account of it's a public event
	for _, event := range res {
		if strings.HasSuffix(event, suffix) || strings.HasSuffix(event, model.PublicAccountHash) {
			// get filters
			filters, err := redis.StringMap(con.Do("HGETALL", getFilterKey(event, pipeline)))
			if err != nil && err != redis.ErrNil {
				lg.WithError(err).Error("error getting trigger filter")
				return nil, err
			}
			// populate trigger object
			trigger := model.Trigger{
				Event:    event,
				Pipeline: pipeline,
				Filters:  filters,
			}
			// get event object, if asked
			if withEvent {
				eventData, err := r.GetEvent(ctx, event)
				if err != nil {
					lg.WithField("event-uri", event).WithError(err).Error("error getting event details")
					return nil, err
				}
				trigger.EventData = *eventData
			}
			// add trigger to result list
			triggers = append(triggers, trigger)
		}
	}
	if len(triggers) == 0 {
		lg.WithField("pipeline", pipeline).Warn("failed to find triggers for pipeline")
	}
	return triggers, nil
}

// DeleteTrigger delete trigger: unlink event from pipeline
func (r *RedisStore) DeleteTrigger(ctx context.Context, event, pipeline string) error {
	account := getAccount(ctx)
	lg := log.WithFields(getContextLogFields(ctx))
	lg.WithFields(log.Fields{
		"pipeline": pipeline,
		"event":    event,
		"account":  account,
	}).Debug("deleting trigger")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()

	// check event account match: public or private
	if !model.MatchPublicAccount(event) && !model.MatchAccount(account, event) {
		lg.WithField("event", event).Error("failed to match trigger for trigger-event")
		return model.ErrTriggerNotFound
	}

	// check Codefresh pipeline match; ignore all errors beside "no match"
	_, err := r.pipelineSvc.GetPipeline(ctx, account, pipeline)
	if err == codefresh.ErrPipelineNoMatch {
		lg.WithError(err).Error("attempt to remove pipeline from another account")
		return err
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		lg.WithError(err).Error("failed to start Redis transaction")
		return err
	}

	// remove pipeline from Triggers
	_, err = con.Do("ZREM", getTriggerKey(account, event), pipeline)
	if err != nil {
		return discardOnError(con, err, lg)
	}

	// remove trigger(s) from Pipelines
	_, err = con.Do("ZREM", getPipelineKey(pipeline), event)
	if err != nil {
		return discardOnError(con, err, lg)
	}

	// remove trigger filters if any
	_, err = con.Do("DEL", getFilterKey(event, pipeline))
	if err != nil {
		return discardOnError(con, err, lg)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		lg.WithError(err).Error("Failed to execute transaction")
	}
	return err
}

// CreateTrigger create trigger: link event <-> multiple pipelines
func (r *RedisStore) CreateTrigger(ctx context.Context, event, pipeline string, filters map[string]string) error {
	account := getAccount(ctx)
	lg := log.WithFields(getContextLogFields(ctx))
	lg.WithFields(log.Fields{
		"event":    event,
		"pipeline": pipeline,
		"account":  account,
		"filters":  filters,
	}).Debug("Creating triggers")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()

	// check event account match: public or private
	if !model.MatchPublicAccount(event) && !model.MatchAccount(account, event) {
		lg.WithField("event", event).Error("failed to match trigger for trigger-event")
		return model.ErrTriggerNotFound
	}

	// check Codefresh pipeline existence
	_, err := r.pipelineSvc.GetPipeline(ctx, account, pipeline)
	if err != nil {
		lg.WithError(err).Error("failed to get pipelines")
		return err
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		lg.WithError(err).Error("failed to start Redis transaction")
		return err
	}

	// add trigger to Pipelines
	_, err = con.Do("ZADD", getPipelineKey(pipeline), 0, event)
	if err != nil {
		return discardOnError(con, err, lg)
	}

	// add pipeline to Triggers
	_, err = con.Do("ZADD", getTriggerKey(account, event), 0, pipeline)
	if err != nil {
		return discardOnError(con, err, lg)
	}

	// add trigger filters to Filters
	if filters != nil {
		filterKey := getFilterKey(event, pipeline)
		for k, v := range filters {
			if _, err = con.Do("HSET", filterKey, k, v); err != nil {
				return discardOnError(con, err, lg)
			}
		}
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		lg.WithError(err).Error("failed to execute transaction")
	}
	return err
}

// GetTriggerPipelines get pipelines that have trigger defined with filter applied
// can be filtered by event-uri(s)
func (r *RedisStore) GetTriggerPipelines(ctx context.Context, event string, vars map[string]string) ([]string, error) {
	account := getAccount(ctx)
	lg := log.WithFields(getContextLogFields(ctx))
	lg.WithFields(log.Fields{
		"event":   event,
		"account": account,
		"vars":    vars,
	}).Debug("getting pipelines for trigger event")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()
	// check trigger existence
	exists, err := redis.Int(con.Do("EXISTS", getTriggerKey(account, event)))
	if err != nil {
		lg.WithError(err).Error("failed to check trigger existence")
		return nil, err
	}
	// if trigger does not exists
	if exists == 0 {
		lg.Warn("trigger not found")
		return nil, nil
	}
	// get pipelines from Triggers
	pipelines, err := redis.Strings(con.Do("ZRANGE", getTriggerKey(account, event), 0, -1))
	if err != nil {
		lg.WithError(err).Error("error getting pipelines")
		return nil, err
	}

	// scan through pipelines and filter out pipelines that match filter
	if vars != nil && len(vars) > 0 {
		skipPipelines := make([]string, 0)
		for _, pipeline := range pipelines {
			// get hash values
			filters, err := redis.StringMap(con.Do("HGETALL", getFilterKey(event, pipeline)))
			if err != nil && err != redis.ErrNil {
				lg.WithError(err).Error("error getting trigger filter")
				return nil, err
			}
			// filter out pipelines: for each filter condition find match
			for name, exp := range filters {
				// get matched value from vars
				val := vars[name]
				// for non-empty value validate
				if val != "" {
					lg.WithFields(log.Fields{
						"name":       name,
						"value":      val,
						"expression": exp,
					}).Debug("filtering pipeline based on value")
					// handle NOT on regex
					skipMatch := false
					if strings.HasPrefix(exp, "SKIP:") {
						exp = strings.TrimPrefix(exp, "SKIP:")
						skipMatch = true
					}
					r, err := regexp.Compile(exp)
					if err != nil {
						lg.WithFields(log.Fields{
							"name":       name,
							"expression": exp,
						}).Error("bad regex expression for filter")
						continue // skip
					}
					// skip matching values (or not)
					if r.MatchString(val) == skipMatch {
						lg.WithField("pipeline", pipeline).Debug("skipping pipeline with filter")
						skipPipelines = append(skipPipelines, pipeline)
					}
				}
			}
		}
		// remove pipelines that match filter
		pipelines = util.DiffStrings(pipelines, skipPipelines)
	}

	if len(pipelines) == 0 {
		lg.Warn("no pipelines found or all skipped")
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
	lg := log.WithFields(getContextLogFields(ctx))
	lg.WithFields(log.Fields{
		"type":    eventType,
		"kind":    kind,
		"values":  values,
		"account": account,
	}).Debug("Creating a new trigger event")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()
	// construct event URI
	eventURI, err := r.eventProvider.ConstructEventURI(eventType, kind, account, values)
	if err != nil {
		lg.WithError(err).Error("failed to create valid event uri")
		return nil, err
	}

	// first, try to get existing event, continue on error
	if event, e := r.GetEvent(ctx, eventURI); e == nil {
		lg.WithField("event-uri", eventURI).Debug("event already exists, reusing trigger-event")
		return event, nil
	}

	// generate random secret if required
	if secret == model.GenerateKeyword {
		lg.Debug("auto generating trigger secret")
		secret = util.RandomString(16)
	}

	// get credentials from Codefresh context (simple key:value map)
	var credentials map[string]string
	if context != "" {
		err = json.Unmarshal([]byte(context), &credentials)
		if err != nil {
			lg.WithError(err).WithField("context", context).Warning("failed to get credentials from context")
		}
	}

	// try subscribing to event - create event in remote system through event provider
	eventInfo, err := r.eventProvider.SubscribeToEvent(ctx, eventURI, secret, credentials)
	if err != nil {
		if err == provider.ErrNotImplemented {
			lg.Warn("event-provider does not implement SubscribeToEvent method")
			lg.Debug("fallback to GetEventInfo method")
			// try to get event info (required method)
			eventInfo, err = r.eventProvider.GetEventInfo(ctx, eventURI, secret)
			if err != nil {
				lg.WithError(err).Error("failed to get event info from event provider")
				return nil, err
			}
		} else {
			lg.WithError(err).Error("failed to subscribe to event in event provider")
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
		lg.WithError(err).Error("failed to start Redis transaction")
		return nil, err
	}
	// store event type
	if _, err := con.Do("HSETNX", eventKey, "type", eventType); err != nil {
		return nil, discardOnError(con, err, lg)
	}
	// store event kind
	if _, err := con.Do("HSETNX", eventKey, "kind", kind); err != nil {
		return nil, discardOnError(con, err, lg)
	}
	// store event account, if not empty
	if _, err := con.Do("HSETNX", eventKey, "account", account); err != nil {
		return nil, discardOnError(con, err, lg)
	}
	// store event secret
	if _, err := con.Do("HSETNX", eventKey, "secret", secret); err != nil {
		return nil, discardOnError(con, err, lg)
	}
	// store event description (from event provider)
	if _, err := con.Do("HSETNX", eventKey, "description", eventInfo.Description); err != nil {
		return nil, discardOnError(con, err, lg)
	}
	// store event endpoint (from event provider)
	if _, err := con.Do("HSETNX", eventKey, "endpoint", eventInfo.Endpoint); err != nil {
		return nil, discardOnError(con, err, lg)
	}
	// store event help (from event provider)
	if _, err := con.Do("HSETNX", eventKey, "help", eventInfo.Help); err != nil {
		return nil, discardOnError(con, err, lg)
	}
	// store event status (from event provider)
	if _, err := con.Do("HSETNX", eventKey, "status", eventInfo.Status); err != nil {
		return nil, discardOnError(con, err, lg)
	}
	// submit transaction
	if _, err := con.Do("EXEC"); err != nil {
		lg.WithError(err).Error("failed to execute transaction")
		return nil, err
	}
	return &event, nil
}

// GetEvent get event by event URI
func (r *RedisStore) GetEvent(ctx context.Context, event string) (*model.Event, error) {
	return r.eventGetter.GetEvent(ctx, event)
}

// GetEvent get event by event URI
func (r *RedisEventGetter) GetEvent(ctx context.Context, event string) (*model.Event, error) {
	account := getAccount(ctx)
	lg := log.WithFields(getContextLogFields(ctx))
	lg.WithFields(log.Fields{
		"event-uri": event,
		"account":   account,
	}).Debug("getting trigger event")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()
	// prepare key
	eventKey := getEventKey(account, event)
	// check event URI is a single event key
	n, err := redis.Int(con.Do("EXISTS", eventKey))
	if err != nil {
		lg.WithError(err).Error("failed to check trigger event existence")
		return nil, err
	}
	if n != 1 {
		lg.Error("trigger event key does not exist")
		return nil, model.ErrEventNotFound
	}

	// get hash values
	fields, err := redis.StringMap(con.Do("HGETALL", eventKey))
	if err != nil {
		lg.WithError(err).Error("failed to get trigger event fields")
		return nil, err
	}
	if len(fields) == 0 {
		lg.Error("failed to find trigger event")
		return nil, model.ErrEventNotFound
	}
	return model.StringsMapToEvent(event, fields), nil
}

// GetEvents get events by event type, kind and filter (can be URI or part of URI)
func (r *RedisStore) GetEvents(ctx context.Context, eventType, kind, filter string) ([]model.Event, error) {
	account := getAccount(ctx)
	public := getPublicFlag(ctx)
	lg := log.WithFields(getContextLogFields(ctx))
	lg.WithFields(log.Fields{
		"type":    eventType,
		"kind":    kind,
		"account": account,
		"filter":  filter,
		"public":  public,
	}).Debug("getting trigger events")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()
	// get all events URIs for account
	uris, err := redis.Strings(con.Do("KEYS", getEventKey(account, filter)))
	if err != nil {
		lg.WithError(err).Error("failed to get trigger events")
		return nil, err
	}
	// get public trigger events, if asked (through context)
	if public {
		// get all events URIs for account
		publicURIs, err := redis.Strings(con.Do("KEYS", getEventKey(model.PublicAccount, filter)))
		if err != nil && err != redis.ErrNil {
			lg.WithError(err).Error("failed to get public trigger events")
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
			lg.WithError(err).Error("failed to get event")
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
	lg := log.WithFields(getContextLogFields(ctx))
	log.WithFields(log.Fields{
		"event-uri": event,
		"account":   account,
	}).Debug("deleting trigger event")
	// record NewRelic segment
	if txn := getNewRelicTransaction(ctx); txn != nil {
		s := newrelic.StartSegment(txn, util.GetCurrentFuncName())
		defer s.End()
	}
	// get redis connection
	con := r.redisPool.GetConn()
	// prepare keys
	eventKey := getEventKey(account, event)
	triggerKey := getTriggerKey(account, event)
	// check event URI is a single event key
	n, err := redis.Int(con.Do("EXISTS", eventKey))
	if err != nil {
		lg.WithError(err).Error("failed to check trigger event existence")
		return err
	}
	if n == 0 {
		lg.Error("trigger event key does not exist")
		return model.ErrEventNotFound
	}
	// check trigger event account vs passed account; skip 'public' events
	a, err := redis.String(con.Do("HGET", eventKey, "account"))
	if err != nil && err != redis.ErrNil {
		lg.WithError(err).Error("failed to get trigger event account")
		return err
	}
	// if not public and belongs to different account - return not exists error
	if a != model.PublicAccount && a != account {
		lg.Error("trigger event account does not match")
		return model.ErrEventNotFound
	}

	// get pipelines linked to the trigger event
	pipelines, err := redis.Strings(con.Do("ZRANGE", triggerKey, 0, -1))
	if err != nil {
		lg.WithError(err).Error("failed to get pipelines for the trigger event")
		return err
	}

	// abort delete operation if trigger event has linked pipelines
	if len(pipelines) > 0 {
		lg.Error("there are triggers linked to this trigger-event, first delete triggers")
		return model.ErrEventDeleteWithTriggers
	}

	// start Redis transaction
	_, err = con.Do("MULTI")
	if err != nil {
		lg.WithError(err).Error("failed to start Redis transaction")
		return err
	}

	// delete event hash for key
	lg.Debug("removing trigger event")
	_, err = con.Do("DEL", eventKey)
	if err != nil {
		lg.WithError(err).Error("failed to delete trigger event")
		return discardOnError(con, err, lg)
	}

	// delete trigger event from Triggers
	lg.Debug("removing trigger event from Triggers")
	_, err = con.Do("DEL", triggerKey)
	if err != nil {
		return discardOnError(con, err, lg)
	}

	// submit transaction
	_, err = con.Do("EXEC")
	if err != nil {
		lg.WithError(err).Error("failed to execute transaction")
		return err
	}

	// get credentials from Codefresh context (simple key:value map)
	var credentials map[string]string
	if context != "" {
		err = json.Unmarshal([]byte(context), &credentials)
		if err != nil {
			lg.WithError(err).WithField("context", context).Warning("failed to get credentials from context")
		}
	}

	// try unsubscribing from event - delete event in remote system through event provider
	err = r.eventProvider.UnsubscribeFromEvent(ctx, event, credentials)
	if err != nil {
		if err != provider.ErrNotImplemented {
			lg.WithError(err).Error("failed to UnsubscribeFromEven")
			return err
		}
		lg.Warn("event provider does not implement UnsubscribeFromEvent method")
	}

	return nil
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
