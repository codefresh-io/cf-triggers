package backend

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
	"github.com/garyburd/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/mock"
)

type RedisPoolMock struct {
	conn *redigomock.Conn
}

func (r *RedisPoolMock) GetConn() redis.Conn {
	if r.conn == nil {
		r.conn = redigomock.NewConn()
	}
	return r.conn
}

// Mock some RedisStore methods

type storeMock struct {
	mock.Mock
}

func (c *storeMock) StoreTrigger(t model.Trigger) error {
	args := c.Called(t)
	return args.Error(0)
}

func (c *storeMock) Delete(e string) error {
	args := c.Called(e)
	return args.Error(0)
}

func (c *storeMock) Add(t model.Trigger) error {
	args := c.Called(t)
	return args.Error(0)
}

func (c *storeMock) Get(e string) (*model.Trigger, error) {
	args := c.Called(e)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Trigger), args.Error(1)
}

// helper function to convert []string to []interface{}
// see https://github.com/golang/go/wiki/InterfaceSlice
func interfaceSlice(slice []string) []interface{} {
	islice := make([]interface{}, len(slice))
	for i, v := range slice {
		islice[i] = v
	}
	return islice
}

func Test_getPrefixKey(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"without prefix", "github.com:project:test", "trigger:github.com:project:test"},
		{"with prefix", "trigger:github.com:project:test", "trigger:github.com:project:test"},
		{"empty", "", "trigger:*"},
		{"star", "*", "trigger:*"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTriggerKey(tt.id); got != tt.want {
				t.Errorf("getKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_List(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	tests := []struct {
		name      string
		fields    fields
		filter    string
		keys      []string
		pipelines [][]string
		want      []*model.Trigger
		wantErr   bool
	}{
		{
			"get empty list",
			fields{redisPool: &RedisPoolMock{}},
			"",
			[]string{},
			[][]string{},
			[]*model.Trigger{},
			false,
		},
		{
			"get all",
			fields{redisPool: &RedisPoolMock{}},
			"*",
			[]string{"test:1", "test:2"},
			[][]string{
				[]string{"puid-1", "puid-2"},
				[]string{"puid-3", "puid-4"},
			},
			[]*model.Trigger{
				&model.Trigger{Event: "test:1", Secret: "secretA", Pipelines: []string{
					"puid-1", "puid-2",
				}},
				&model.Trigger{Event: "test:2", Secret: "secretB", Pipelines: []string{
					"puid-3", "puid-4",
				}},
			},
			false,
		},
		{
			"get one",
			fields{redisPool: &RedisPoolMock{}},
			"test:*",
			[]string{"test:1"},
			[][]string{
				[]string{"puid-1"},
			},
			[]*model.Trigger{
				&model.Trigger{Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1"}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &storeMock{}
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
				getFunc:   mock.Get,
			}
			r.redisPool.GetConn().(*redigomock.Conn).Command("KEYS", getTriggerKey(tt.filter)).Expect(interfaceSlice(tt.keys))
			for i, k := range tt.keys {
				mock.On("Get", k).Return(tt.want[i], nil)
			}
			got, err := r.List(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.List() = %v, want %v", got, tt.want)
			}
			mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_ListByPipeline(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	type args struct {
		pipelineUID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		events  []string
		want    []*model.Trigger
		wantErr bool
	}{
		{
			"get empty list",
			fields{redisPool: &RedisPoolMock{}},
			args{"puid-1"},
			[]string{},
			[]*model.Trigger{},
			false,
		},
		{
			"get triggers",
			fields{redisPool: &RedisPoolMock{}},
			args{"puid-1"},
			[]string{"test:1", "test:2"},
			[]*model.Trigger{
				&model.Trigger{Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"}},
				&model.Trigger{Event: "test:2", Secret: "secretB", Pipelines: []string{"puid-1", "puid-2"}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &storeMock{}
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
				getFunc:   mock.Get,
			}
			r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getPipelineKey(tt.args.pipelineUID), 0, -1).Expect(interfaceSlice(tt.events))
			for i, eventURI := range tt.events {
				mock.On("Get", eventURI).Return(tt.want[i], nil)
			}
			got, err := r.ListByPipeline(tt.args.pipelineUID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.ListByPipeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.ListByPipeline() = %v, want %v", got, tt.want)
			}
			mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_Get(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	tests := []struct {
		name     string
		fields   fields
		expected []string
		want     *model.Trigger
		wantErr  bool
	}{
		{
			"get trigger by id",
			fields{redisPool: &RedisPoolMock{}},
			[]string{"puid-1", "puid-2", "puid-3"},
			&model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2", "puid-3"},
			},
			false,
		},
		{
			"get trigger GET error",
			fields{redisPool: &RedisPoolMock{}},
			[]string{"puid-1"},
			&model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1"},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
			}
			if tt.wantErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(tt.want.Event)).ExpectError(fmt.Errorf("GET error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(tt.want.Event)).Expect(tt.want.Secret)
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.want.Event), 0, -1).Expect(interfaceSlice(tt.expected))
			}
			got, err := r.Get(tt.want.Event)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_StoreTrigger(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	tests := []struct {
		name    string
		fields  fields
		trigger model.Trigger
		wantErr [3]bool
	}{
		{
			"store trigger",
			fields{redisPool: &RedisPoolMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"},
			},
			[3]bool{false, false, false},
		},
		{
			"store trigger with auto-generated secret",
			fields{redisPool: &RedisPoolMock{}},
			model.Trigger{
				Event: "test:1", Secret: model.GenerateKeyword, Pipelines: []string{"puid-1", "puid-2"},
			},
			[3]bool{false, false, false},
		},
		{
			"store trigger SET error",
			fields{redisPool: &RedisPoolMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"},
			},
			[3]bool{true, false, false},
		},
		{
			"store trigger non-existing pipeline",
			fields{redisPool: &RedisPoolMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"},
			},
			[3]bool{false, true, false},
		},
		{
			"store trigger SADD error",
			fields{redisPool: &RedisPoolMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"},
			},
			[3]bool{false, false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := codefresh.NewCodefreshMockEndpoint()
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: mock,
			}
			// expect Redis transaction open
			r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI").Expect("OK!")
			// error cases
			if tt.wantErr[0] {
				r.redisPool.GetConn().(*redigomock.Conn).Command("SET", getSecretKey(tt.trigger.Event), tt.trigger.Secret).ExpectError(fmt.Errorf("SET error"))
				// expect transaction discard on error
				r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
			} else {
				if tt.trigger.Secret == model.GenerateKeyword {
					r.redisPool.GetConn().(*redigomock.Conn).Command("SET", getSecretKey(tt.trigger.Event), util.TestRandomString).Expect("OK!")
				} else {
					r.redisPool.GetConn().(*redigomock.Conn).Command("SET", getSecretKey(tt.trigger.Event), tt.trigger.Secret).Expect("OK!")
				}
				for _, p := range tt.trigger.Pipelines {
					if tt.wantErr[1] {
						mock.On("CheckPipelineExists", p).Return(false, codefresh.ErrPipelineNotFound)
						// expect transaction discard on error
						r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
						break
					} else {
						mock.On("CheckPipelineExists", p).Return(true, nil)
					}
					if tt.wantErr[2] {
						r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getTriggerKey(tt.trigger.Event), 0, p).ExpectError(fmt.Errorf("ZADD error"))
						// expect transaction discard on error
						r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
						break
					} else {
						r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getTriggerKey(tt.trigger.Event), 0, p)
					}
					r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getPipelineKey(p), 0, tt.trigger.Event)
					// expect Redis transaction exec
					r.redisPool.GetConn().(*redigomock.Conn).Command("EXEC").Expect("OK!")
				}
			}
			// perform function call
			if err := r.StoreTrigger(tt.trigger); (err != nil) != (tt.wantErr[0] || tt.wantErr[1] || tt.wantErr[2]) {
				t.Errorf("RedisStore.storeTrigger() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert expectation
			mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_Add(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	tests := []struct {
		name    string
		fields  fields
		trigger model.Trigger
		mock    *storeMock
		count   int64 // number of existing pipelines for trigger
		wantErr error
	}{
		{
			"add trigger",
			fields{redisPool: &RedisPoolMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"},
			},
			&storeMock{},
			0,
			nil,
		},
		{
			"try to add existing trigger",
			fields{redisPool: &RedisPoolMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"},
			},
			&storeMock{},
			10,
			model.ErrTriggerAlreadyExists,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:        tt.fields.redisPool,
				storeTriggerFunc: tt.mock.StoreTrigger,
			}
			// mock redis
			r.redisPool.GetConn().(*redigomock.Conn).Command("ZCARD", getTriggerKey(tt.trigger.Event)).Expect(int64(tt.count))
			// mock store call
			if tt.count == 0 {
				tt.mock.On("StoreTrigger", tt.trigger).Return(nil)
			}
			// if tt.count == 0 { // following commands run only if there is no existing trigger with pipelines
			if err := r.Add(tt.trigger); err != nil && err != tt.wantErr {
				t.Errorf("RedisStore.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert expectation
			tt.mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_Update(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	tests := []struct {
		name      string
		fields    fields
		pipelines []string
		trigger   model.Trigger
		mock      *storeMock
		count     int64
		wantErr   error
	}{
		{
			"update trigger",
			fields{redisPool: &RedisPoolMock{}},
			[]string{"puid-1", "puid-2"},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"},
			},
			&storeMock{},
			1,
			nil,
		},
		{
			"try to update non existing trigger",
			fields{redisPool: &RedisPoolMock{}},
			[]string{"puid-1", "puid-2"},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []string{"puid-1", "puid-2"},
			},
			&storeMock{},
			0,
			model.ErrTriggerNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:        tt.fields.redisPool,
				storeTriggerFunc: tt.mock.StoreTrigger,
			}
			// mock redis
			r.redisPool.GetConn().(*redigomock.Conn).Command("ZCARD", getTriggerKey(tt.trigger.Event)).Expect(tt.count)
			if tt.wantErr != nil {
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.trigger.Event), 0, -1).Expect(nil)
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.trigger.Event), 0, -1).Expect(interfaceSlice(tt.pipelines))
				// mock store call
				tt.mock.On("StoreTrigger", tt.trigger).Return(nil)
			}
			// if tt.count == 0 { // following commands run only if there is no existing trigger with pipelines
			if err := r.Update(tt.trigger); err != nil && err != tt.wantErr {
				t.Errorf("RedisStore.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert expectation
			tt.mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_Delete(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	type args struct {
		id        string
		pipelines []string
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		wantDelSecretErr  bool
		wantDelTriggerErr bool
	}{
		{
			"delete trigger",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test", pipelines: []string{"p1", "p2"}},
			false,
			false,
		},
		{
			"delete trigger from single pipeline",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test", pipelines: []string{"p1"}},
			false,
			false,
		},
		{
			"delete trigger DEL secret error",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test"},
			true,
			false,
		},
		{
			"delete trigger DEL trigger error",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test"},
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
			}
			// get pipelines
			r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.args.id), 0, -1).Expect(interfaceSlice(tt.args.pipelines))
			// expect Redis transaction open
			r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI").Expect("OK!")
			// delete secret
			if tt.wantDelSecretErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", getSecretKey(tt.args.id)).ExpectError(fmt.Errorf("DEL STRING error"))
				// expect transaction discard on error
				r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", getSecretKey(tt.args.id)).Expect("OK!")
			}
			// delete trigger from pipelines
			for _, p := range tt.args.pipelines {
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", getPipelineKey(p), tt.args.id)
			}
			// delete trigger
			if tt.wantDelTriggerErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", getTriggerKey(tt.args.id)).ExpectError(fmt.Errorf("DEL SET error"))
				// expect transaction discard on error
				r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", getTriggerKey(tt.args.id)).Expect("OK!")
				r.redisPool.GetConn().(*redigomock.Conn).Command("EXEC").Expect("OK!")
			}
			if err := r.Delete(tt.args.id); (err != nil) != (tt.wantDelSecretErr || tt.wantDelTriggerErr) {
				t.Errorf("RedisStore.Delete() error = %v", err)
			}
		})
	}
}

func TestMain(m *testing.M) {
	util.TestMode = true
	os.Exit(m.Run())
}

func TestRedisStore_Ping(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			"happy ping",
			fields{redisPool: &RedisPoolMock{}},
			"PONG",
			false,
		},
		{
			"failed ping - no Redis",
			fields{redisPool: &RedisPoolMock{}},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
			}
			if tt.wantErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("PING").ExpectError(fmt.Errorf("PING error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("PING").Expect(tt.want)
			}
			got, err := r.Ping()
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.Ping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RedisStore.Ping() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_GetPipelines(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	type args struct {
		id        string
		pipelines []string
	}
	tests := []struct {
		name         string
		fields       fields
		args         []args
		expected     []string
		wantRedisErr bool
		wantEmptyErr bool
	}{
		{
			"get single trigger pipelines",
			fields{redisPool: &RedisPoolMock{}},
			[]args{
				{
					id:        "event:test:uri",
					pipelines: []string{"puid-1", "puid-2", "puid-3"},
				},
			},
			[]string{"puid-1", "puid-2", "puid-3"},
			false,
			false,
		},
		{
			"get multi trigger pipelines",
			fields{redisPool: &RedisPoolMock{}},
			[]args{
				{
					id:        "event:test-1:uri",
					pipelines: []string{"puid-1", "puid-2", "puid-3"},
				},
				{
					id:        "event:test-2:uri",
					pipelines: []string{"puid-4", "puid-5", "puid-6"},
				},
			},
			[]string{"puid-1", "puid-2", "puid-3", "puid-4", "puid-5", "puid-6"},
			false,
			false,
		},
		{
			"get multi trigger pipelines (duplicate)",
			fields{redisPool: &RedisPoolMock{}},
			[]args{
				{
					id:        "event:test-1:uri",
					pipelines: []string{"puid-1", "puid-2", "puid-3"},
				},
				{
					id:        "event:test-2:uri",
					pipelines: []string{"puid-2", "puid-3", "puid-4"},
				},
			},
			[]string{"puid-1", "puid-2", "puid-3", "puid-4"},
			false,
			false,
		},
		{
			"get all pipelines",
			fields{redisPool: &RedisPoolMock{}},
			[]args{
				{
					id:        "",
					pipelines: []string{"puid-1", "puid-2", "puid-3", "puid-4", "puid-5", "puid-6"},
				},
			},
			[]string{"puid-1", "puid-2", "puid-3", "puid-4", "puid-5", "puid-6"},
			false,
			false,
		},
		{
			"get trigger pipelines ZRANGE error",
			fields{redisPool: &RedisPoolMock{}},
			[]args{
				{
					id:        "event:test:uri",
					pipelines: nil,
				},
			},
			nil,
			true,
			false,
		},
		{
			"get trigger pipelines EMPTY error",
			fields{redisPool: &RedisPoolMock{}},
			[]args{
				{
					id:        "event:test:uri",
					pipelines: nil,
				},
			},
			nil,
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
			}
			if tt.wantRedisErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.args[0].id), 0, -1).ExpectError(fmt.Errorf("ZRANGE error"))
			} else if tt.wantEmptyErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.args[0].id), 0, -1).Expect(interfaceSlice(tt.args[0].pipelines))
			} else {
				for _, _arg := range tt.args {
					if _arg.id == "" {
						r.redisPool.GetConn().(*redigomock.Conn).Command("KEYS", getPipelineKey(_arg.id)).Expect(interfaceSlice(_arg.pipelines))
					} else {
						r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(_arg.id), 0, -1).Expect(interfaceSlice(_arg.pipelines))
					}
				}
			}
			var ids []string
			for _, _arg := range tt.args {
				if _arg.id != "" {
					ids = append(ids, _arg.id)
				}
			}
			got, err := r.GetPipelines(ids)
			if (err != nil) != (tt.wantRedisErr || tt.wantEmptyErr) {
				t.Errorf("RedisStore.GetPipelines() error = %v, wantErr %v", err, (tt.wantRedisErr || tt.wantEmptyErr))
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("RedisStore.GetPipelines() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRedisStore_GetSecret(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	type args struct {
		eventURI string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "get secret",
			fields:  fields{redisPool: &RedisPoolMock{}},
			args:    args{eventURI: "event:test"},
			want:    "123456789",
			wantErr: false,
		},
		{
			name:    "get secret missing",
			fields:  fields{redisPool: &RedisPoolMock{}},
			args:    args{eventURI: "event:test"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
			}
			if tt.wantErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(tt.args.eventURI)).ExpectError(fmt.Errorf("GET error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(tt.args.eventURI)).Expect(tt.want)
			}
			got, err := r.GetSecret(tt.args.eventURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RedisStore.GetSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_CreateTriggersForPipeline(t *testing.T) {
	type redisErrors struct {
		multi bool
		zadd1 bool
		zadd2 bool
		exec  bool
	}
	type args struct {
		pipeline string
		events   []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		errs    redisErrors
	}{
		{
			"create pipeline triggers for multiple events",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"create trigger for non-existing pipeline",
			args{
				pipeline: "non-existing-pipeline",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{false, false, false, false},
		},
		{
			"fail start transaction",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{true, false, false, false},
		},
		{
			"fail adding pipeline to Triggers map",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{false, true, false, false},
		},
		{
			"fail adding events to Pipelines map",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{false, false, true, false},
		},
		{
			"fail exec transaction",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{false, false, false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := codefresh.NewCodefreshMockEndpoint()
			r := &RedisStore{
				redisPool:   &RedisPoolMock{},
				pipelineSvc: mock,
			}
			var cmd *redigomock.Cmd

			// mock Codefresh API call
			if tt.args.pipeline == "non-existing-pipeline" {
				mock.On("CheckPipelineExists", tt.args.pipeline).Return(false, codefresh.ErrPipelineNotFound)
				goto Invoke
			} else {
				mock.On("CheckPipelineExists", tt.args.pipeline).Return(true, nil)
			}

			// expect Redis transaction open
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI")
			if tt.errs.multi {
				cmd.ExpectError(errors.New("MULTI error"))
				goto Invoke
			} else {
				cmd.Expect("OK!")
			}
			// add pipeline to event(s)
			for _, event := range tt.args.events {
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getTriggerKey(event), 0, tt.args.pipeline)
				if tt.errs.zadd1 {
					cmd.ExpectError(errors.New("ZADD error"))
					goto EndTransaction
				}
			}
			// add events to the Pipelines map
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getPipelineKey(tt.args.pipeline), 0, tt.args.events)
			if tt.errs.zadd2 {
				cmd.ExpectError(errors.New("ZADD error"))
			}

		EndTransaction:
			// discard transaction on error
			if tt.wantErr && !tt.errs.exec {
				// expect transaction discard on error
				r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
			} else {
				// expect Redis transaction exec
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("EXEC")
				if tt.errs.exec {
					cmd.ExpectError(errors.New("EXEC error"))
				} else {
					cmd.Expect("OK!")
				}
			}

		Invoke:
			// invoke method
			if err := r.CreateTriggersForPipeline(tt.args.pipeline, tt.args.events); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.CreateTriggersForPipeline() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert mock
			mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_DeleteTriggersForPipeline(t *testing.T) {
	type redisErrors struct {
		multi bool
		zrem1 bool
		zrem2 bool
		exec  bool
	}
	type args struct {
		pipeline string
		events   []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		errs    redisErrors
	}{
		{
			"delete triggers for pipeline",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"delete single event trigger for pipeline",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"fail start transaction",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{true, false, false, false},
		},
		{
			"fail remove pipeline from Triggers map",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{false, true, false, false},
		},
		{
			"fail remove events from Pipelines map",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{false, false, true, false},
		},
		{
			"fail exec transaction",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"event:uri:test:1", "event:uri:test:2"},
			},
			true,
			redisErrors{false, false, false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
			}
			// expect Redis transaction open
			cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI")
			if tt.errs.multi {
				cmd.ExpectError(errors.New("MULTI error"))
				goto Invoke
			} else {
				cmd.Expect("OK!")
			}
			// add pipeline to event(s)
			for _, event := range tt.args.events {
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", getTriggerKey(event), tt.args.pipeline)
				if tt.errs.zrem1 {
					cmd.ExpectError(errors.New("ZREM error"))
					goto EndTransaction
				}
			}
			// add events to the Pipelines map
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", getPipelineKey(tt.args.pipeline), tt.args.events)
			if tt.errs.zrem2 {
				cmd.ExpectError(errors.New("ZREM error"))
			}

		EndTransaction:
			// discard transaction on error
			if tt.wantErr && !tt.errs.exec {
				// expect transaction discard on error
				r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
			} else {
				// expect Redis transaction exec
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("EXEC")
				if tt.errs.exec {
					cmd.ExpectError(errors.New("EXEC error"))
				} else {
					cmd.Expect("OK!")
				}
			}

			// invoke method
		Invoke:
			if err := r.DeleteTriggersForPipeline(tt.args.pipeline, tt.args.events); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.DeleteTriggersForPipeline() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisStore_CreateTriggersForEvent(t *testing.T) {
	type redisErrors struct {
		multi bool
		zadd1 bool
		zadd2 bool
		exec  bool
	}
	type args struct {
		event     string
		pipelines []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		errs    redisErrors
	}{
		{
			"create event trigger for multiple pipelines",
			args{
				event:     "event:uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"create event trigger for non-existing pipeline",
			args{
				event:     "event:uri:test",
				pipelines: []string{"non-existing-pipeline", "owner:repo:test"},
			},
			true,
			redisErrors{false, false, false, false},
		},
		{
			"fail start transaction",
			args{
				event:     "event:uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			true,
			redisErrors{true, false, false, false},
		},
		{
			"fail adding pipeline to Triggers map",
			args{
				event:     "event:uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			true,
			redisErrors{false, true, false, false},
		},
		{
			"fail adding events to Pipelines map",
			args{
				event:     "event:uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			true,
			redisErrors{false, false, true, false},
		},
		{
			"fail exec transaction",
			args{
				event:     "event:uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			true,
			redisErrors{false, false, false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := codefresh.NewCodefreshMockEndpoint()
			r := &RedisStore{
				redisPool:   &RedisPoolMock{},
				pipelineSvc: mock,
			}
			// expect Redis transaction open
			cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI")
			if tt.errs.multi {
				cmd.ExpectError(errors.New("MULTI error"))
				goto Invoke
			} else {
				cmd.Expect("OK!")
			}
			// add pipeline to event(s)
			for _, pipeline := range tt.args.pipelines {
				// mock Codefresh API call
				if pipeline == "non-existing-pipeline" {
					mock.On("CheckPipelineExists", pipeline).Return(false, codefresh.ErrPipelineNotFound)
					goto Invoke
				} else {
					mock.On("CheckPipelineExists", pipeline).Return(true, nil)
				}
				// add events to the Pipelines map
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getPipelineKey(pipeline), 0, tt.args.event)
				if tt.errs.zadd1 {
					cmd.ExpectError(errors.New("ZADD error"))
					goto EndTransaction
				}
			}
			// add events to the Pipelines map
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getTriggerKey(tt.args.event), 0, tt.args.pipelines)
			if tt.errs.zadd2 {
				cmd.ExpectError(errors.New("ZADD error"))
			}

		EndTransaction:
			// discard transaction on error
			if tt.wantErr && !tt.errs.exec {
				// expect transaction discard on error
				r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
			} else {
				// expect Redis transaction exec
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("EXEC")
				if tt.errs.exec {
					cmd.ExpectError(errors.New("EXEC error"))
				} else {
					cmd.Expect("OK!")
				}
			}

		Invoke:
			if err := r.CreateTriggersForEvent(tt.args.event, tt.args.pipelines); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.CreateTriggersForEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert mock
			mock.AssertExpectations(t)
		})
	}
}
