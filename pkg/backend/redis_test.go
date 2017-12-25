package backend

import (
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

// Mock CF API

type CFMock struct {
	mock.Mock
}

func (c *CFMock) CheckPipelineExist(repoOwner string, repoName string, name string) error {
	args := c.Called(repoOwner, repoName, name)
	return args.Error(0)
}

func (c *CFMock) RunPipeline(repoOwner string, repoName string, name string, vars map[string]string) (string, error) {
	args := c.Called(repoOwner, repoName, name, vars)
	return args.String(0), args.Error(1)
}

func (c *CFMock) Ping() error {
	args := c.Called()
	return args.Error(0)
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
				[]string{"ownerA:repoA:test1", "ownerA:repoA:test2"},
				[]string{"ownerB:repoB:test", "ownerC:repoC:test"},
			},
			[]*model.Trigger{
				&model.Trigger{Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "test1"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "test2"},
				}},
				&model.Trigger{Event: "test:2", Secret: "secretB", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerB", RepoName: "repoB", Name: "test"},
					{RepoOwner: "ownerC", RepoName: "repoC", Name: "test"},
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
				[]string{"ownerA:repoA:test"},
			},
			[]*model.Trigger{
				&model.Trigger{Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "test"},
				}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
			}
			r.redisPool.GetConn().(*redigomock.Conn).Command("KEYS", getTriggerKey(tt.filter)).Expect(interfaceSlice(tt.keys))
			for i, k := range tt.keys {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(k)).Expect(tt.want[i].Secret)
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(k), 0, -1).Expect(interfaceSlice(tt.pipelines[i]))
			}
			got, err := r.List(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.List() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_ListByPipeline(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	type args struct {
		pipelineURI string
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
			args{"codefresh:demo:build"},
			[]string{},
			[]*model.Trigger{},
			false,
		},
		{
			"get triggers",
			fields{redisPool: &RedisPoolMock{}},
			args{"codefresh:demo:build"},
			[]string{"test:1", "test:2"},
			[]*model.Trigger{
				&model.Trigger{Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "codefresh", RepoName: "demo", Name: "build"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "test2"},
				}},
				&model.Trigger{Event: "test:2", Secret: "secretB", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerB", RepoName: "repoB", Name: "test"},
					{RepoOwner: "codefresh", RepoName: "demo", Name: "build"},
				}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
			}
			r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getPipelineKey(tt.args.pipelineURI), 0, -1).Expect(interfaceSlice(tt.events))
			for i, eventURI := range tt.events {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(eventURI)).Expect(tt.want[i].Secret)
				pipelines := make([]string, 0)
				for _, p := range tt.want[i].Pipelines {
					pipelines = append(pipelines, model.PipelineToURI(&p))
				}
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(eventURI), 0, -1).Expect(interfaceSlice(pipelines))
			}
			got, err := r.ListByPipeline(tt.args.pipelineURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.ListByPipeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.ListByPipeline() = %v, want %v", got, tt.want)
			}
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
			[]string{"ownerA:repoA:test", "ownerA:repoA:test2", "ownerA:repoB:test"},
			&model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "test"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "test2"},
					{RepoOwner: "ownerA", RepoName: "repoB", Name: "test"},
				},
			},
			false,
		},
		{
			"get trigger GET error",
			fields{redisPool: &RedisPoolMock{}},
			[]string{"ownerA:repoA:test"},
			&model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "test"},
				},
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
		redisPool   RedisPoolService
		pipelineSvc codefresh.PipelineService
	}
	tests := []struct {
		name    string
		fields  fields
		trigger model.Trigger
		wantErr [3]bool
	}{
		{
			"store trigger",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "pipelineA", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipelineB", RepoOwner: "ownerA", RepoName: "repoB"},
				},
			},
			[3]bool{false, false, false},
		},
		{
			"store trigger with auto-generated secret",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			model.Trigger{
				Event: "test:1", Secret: model.GenerateKeyword, Pipelines: []model.Pipeline{
					{Name: "pipelineA", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipelineB", RepoOwner: "ownerA", RepoName: "repoB"},
				},
			},
			[3]bool{false, false, false},
		},
		{
			"store trigger SET error",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "pipelineA", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipelineB", RepoOwner: "ownerA", RepoName: "repoB"},
				},
			},
			[3]bool{true, false, false},
		},
		{
			"store trigger non-existing pipeline",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "pipelineA", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipelineB", RepoOwner: "ownerA", RepoName: "repoB"},
				},
			},
			[3]bool{false, true, false},
		},
		{
			"store trigger SADD error",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "pipelineA", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipelineB", RepoOwner: "ownerA", RepoName: "repoB"},
				},
			},
			[3]bool{false, false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			// mock CF API call
			mock := r.pipelineSvc.(*CFMock)
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
						mock.On("CheckPipelineExist", p.RepoOwner, p.RepoName, p.Name).Return(codefresh.ErrPipelineNotFound)
						// expect transaction discard on error
						r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
						break
					} else {
						mock.On("CheckPipelineExist", p.RepoOwner, p.RepoName, p.Name).Return(nil)
					}
					pipelineURI := model.PipelineToURI(&p)
					if tt.wantErr[2] {
						r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getTriggerKey(tt.trigger.Event), 0, pipelineURI).ExpectError(fmt.Errorf("SADD error"))
						// expect transaction discard on error
						r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
						break
					} else {
						r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getTriggerKey(tt.trigger.Event), 0, pipelineURI)
					}
					r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", getPipelineKey(pipelineURI), 0, tt.trigger.Event)
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
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "pipelineA", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipelineB", RepoOwner: "ownerA", RepoName: "repoB"},
				},
			},
			&storeMock{},
			0,
			nil,
		},
		{
			"try to add existing trigger",
			fields{redisPool: &RedisPoolMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "pipelineA", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipelineB", RepoOwner: "ownerA", RepoName: "repoB"},
				},
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
			[]string{"ownerA:repoA:pipelineA", "ownerA:repoB:pipelineB"},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipelineA"},
					{RepoOwner: "ownerA", RepoName: "repoB", Name: "pipelineB"},
				},
			},
			&storeMock{},
			1,
			nil,
		},
		{
			"try to update non existing trigger",
			fields{redisPool: &RedisPoolMock{}},
			[]string{"ownerA:repoA:pipelineA", "ownerA:repoB:pipelineB"},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipelineA"},
					{RepoOwner: "ownerA", RepoName: "repoB", Name: "pipelineB"},
				},
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

func TestRedisStore_CheckSecret(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	type args struct {
		id      string
		secret  string
		message string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedSecret string
		wantErr        bool
	}{
		{
			"check secret",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test", secret: "secretAAA", message: "hello world"},
			"secretAAA",
			false,
		},
		{
			"check secret error",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test", secret: "secretAAA", message: "hello world"},
			"secretBBB",
			true,
		},
		{
			"check secret signature sha1",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test", secret: "c61fe17e43c57ac8b18a1cb7b2e9ff666f506fa5", message: "hello world"},
			"secretKey",
			false,
		},
		{
			"check empty secret",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test", secret: "", message: "hello world"},
			"",
			false,
		},
		{
			"check no secret passed",
			fields{redisPool: &RedisPoolMock{}},
			args{id: "test", secret: "", message: "hello world"},
			"secretKey",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: tt.fields.redisPool,
			}
			r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(tt.args.id)).Expect(tt.expectedSecret)
			if err := r.CheckSecret(tt.args.id, tt.args.message, tt.args.secret); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.CheckSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisStore_Run(t *testing.T) {
	type fields struct {
		redisPool   RedisPoolService
		pipelineSvc codefresh.PipelineService
	}
	type args struct {
		id        string
		vars      map[string]string
		pipelines []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		trigger model.Trigger
		want    []string
		wantErr bool
	}{
		{
			"run pipeline",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: new(CFMock)},
			args{
				id:        "test:event",
				vars:      map[string]string{"V1": "AAA", "V2": "BBB"},
				pipelines: []string{"ownerA:repoA:pipeline1", "ownerA:repoA:pipeline2", "ownerA:repoB:pipeline3"},
			},
			model.Trigger{
				Event: "test:event", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline1"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline2"},
					{RepoOwner: "ownerA", RepoName: "repoB", Name: "pipeline3"},
				},
			},
			[]string{"pipeline1_runID", "pipeline2_runID", "pipeline3_runID"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			// mock call
			mock := r.pipelineSvc.(*CFMock)
			for i, p := range tt.trigger.Pipelines {
				mock.On("RunPipeline", p.RepoOwner, p.RepoName, p.Name, tt.args.vars).Return(tt.want[i], nil)
			}
			// mock redis commands
			r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(tt.trigger.Event)).Expect(tt.trigger.Secret)
			r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.trigger.Event), 0, -1).Expect(interfaceSlice(tt.args.pipelines))
			// perform call
			got, err := r.Run(tt.args.id, tt.args.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.Run() = %v, want %v", got, tt.want)
			}
			// assert expectation
			mock.AssertExpectations(t)
		})
	}
}

func TestMain(m *testing.M) {
	util.TestMode = true
	os.Exit(m.Run())
}

func TestRedisStore_Ping(t *testing.T) {
	type fields struct {
		redisPool   RedisPoolService
		pipelineSvc codefresh.PipelineService
	}
	tests := []struct {
		name             string
		fields           fields
		want             string
		wantRedisErr     bool
		wantCodefreshErr bool
	}{
		{
			"happy ping",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: new(CFMock)},
			"PONG",
			false,
			false,
		},
		{
			"failed ping - no Redis",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: new(CFMock)},
			"Redis Error",
			true,
			false,
		},
		{
			"failed ping - no Codefresh",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: new(CFMock)},
			"Codefresh Error",
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			// mock call
			mock := r.pipelineSvc.(*CFMock)
			if tt.wantRedisErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("PING").ExpectError(fmt.Errorf("PING error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("PING").Expect(tt.want)
				if tt.wantCodefreshErr {
					mock.On("Ping").Return(fmt.Errorf("Codefresh Ping Error"))
				} else {
					mock.On("Ping").Return(nil)
				}
			}
			got, err := r.Ping()
			if (err != nil) != (tt.wantRedisErr || tt.wantCodefreshErr) {
				t.Errorf("RedisStore.Ping() error = %v, wantErr %v", err, (tt.wantRedisErr || tt.wantCodefreshErr))
				return
			}
			if got != tt.want {
				t.Errorf("RedisStore.Ping() = %v, want %v", got, tt.want)
			}
			// assert expectation
			mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_GetPipelines(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	tests := []struct {
		name         string
		fields       fields
		id           string
		pipelines    []string
		want         []model.Pipeline
		wantRedisErr bool
		wantEmptyErr bool
	}{
		{
			"get trigger pipelines",
			fields{redisPool: &RedisPoolMock{}},
			"event:test:uri",
			[]string{"ownerA:repoA:test", "ownerA:repoA:testB", "ownerB:repoB:testC"},
			[]model.Pipeline{
				{RepoOwner: "ownerA", RepoName: "repoA", Name: "test"},
				{RepoOwner: "ownerA", RepoName: "repoA", Name: "testB"},
				{RepoOwner: "ownerB", RepoName: "repoB", Name: "testC"},
			},
			false,
			false,
		},
		{
			"get trigger pipelines ZRANGE error",
			fields{redisPool: &RedisPoolMock{}},
			"event:test:uri",
			[]string{},
			nil,
			true,
			false,
		},
		{
			"get trigger pipelines EMPTY error",
			fields{redisPool: &RedisPoolMock{}},
			"event:test:uri",
			[]string{},
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
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.id), 0, -1).ExpectError(fmt.Errorf("ZRANGE error"))
			} else if tt.wantEmptyErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.id), 0, -1).Expect(interfaceSlice(tt.pipelines))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.id), 0, -1).Expect(interfaceSlice(tt.pipelines))
			}
			got, err := r.GetPipelines(tt.id)
			if (err != nil) != (tt.wantRedisErr || tt.wantEmptyErr) {
				t.Errorf("RedisStore.GetPipelines() error = %v, wantErr %v", err, (tt.wantRedisErr || tt.wantEmptyErr))
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.GetPipelines() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_AddPipelines(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	type args struct {
		id        string
		pipelines []model.Pipeline
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		pipelines []string
		trigger   model.Trigger
		mock      *storeMock
		exists    bool
		wantErr   bool
	}{
		{
			"add pipelines to existing trigger",
			fields{redisPool: &RedisPoolMock{}},
			args{
				id: "event:test:uri",
				pipelines: []model.Pipeline{
					{Name: "test", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "testB", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "testC", RepoOwner: "ownerB", RepoName: "repoB"},
				},
			},
			[]string{"ownerA:repoA:pipeline1", "ownerA:repoA:pipeline2"},
			model.Trigger{
				Event: "event:test:uri", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline1"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline2"},
				},
			},
			&storeMock{},
			true,
			false,
		},
		{
			"add pipelines leads to new trigger",
			fields{redisPool: &RedisPoolMock{}},
			args{
				id: "event:test:uri",
				pipelines: []model.Pipeline{
					{Name: "test", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "testB", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "testC", RepoOwner: "ownerB", RepoName: "repoB"},
				},
			},
			[]string{"ownerA:repoA:pipeline1", "ownerA:repoA:pipeline2"},
			model.Trigger{
				Event: "event:test:uri", Secret: "", Pipelines: []model.Pipeline{},
			},
			&storeMock{},
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:        tt.fields.redisPool,
				storeTriggerFunc: tt.mock.StoreTrigger,
			}
			// mock redis calls for Get() command
			if !tt.exists {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(tt.args.id)).Expect(nil)
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.args.id), 0, -1).Expect(nil)
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZCARD", getTriggerKey(tt.args.id)).Expect(nil)
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", getSecretKey(tt.args.id)).Expect(tt.trigger.Secret)
				r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.args.id), 0, -1).Expect(interfaceSlice(tt.pipelines))
			}
			// add pipelines to trigger
			trigger := tt.trigger
			for _, p := range tt.args.pipelines {
				trigger.Pipelines = append(trigger.Pipelines, p)
			}
			// mock store call
			tt.mock.On("StoreTrigger", trigger).Return(nil)
			if err := r.AddPipelines(tt.args.id, tt.args.pipelines); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.AddPipelines() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert expectations
			tt.mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_DeletePipeline(t *testing.T) {
	type fields struct {
		redisPool RedisPoolService
	}
	type args struct {
		id  string
		pid string
	}
	type expected struct {
		remove int64
		remain int64
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		pipelines []string
		existing  *model.Trigger
		mock      *storeMock
		expected  expected
		wantErr   bool
	}{
		{
			"delete pipeline from trigger",
			fields{redisPool: &RedisPoolMock{}},
			args{
				id:  "event:test:uri",
				pid: "ownerA:repoA:pipeline1",
			},
			[]string{"ownerA:repoA:pipeline1", "ownerA:repoA:pipeline2"},
			&model.Trigger{
				Event: "event:test:uri", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline1"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline2"},
				},
			},
			&storeMock{},
			expected{1, 2},
			false,
		},
		{
			"delete last pipeline from trigger -> delete trigger",
			fields{redisPool: &RedisPoolMock{}},
			args{
				id:  "event:test:uri",
				pid: "ownerA:repoA:pipeline1",
			},
			[]string{"ownerA:repoA:pipeline1", "ownerA:repoA:pipeline2"},
			&model.Trigger{
				Event: "event:test:uri", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline1"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline2"},
				},
			},
			&storeMock{},
			expected{1, 0},
			false,
		},
		{
			"try to delete non-existing pipeline from trigger",
			fields{redisPool: &RedisPoolMock{}},
			args{
				id:  "event:test:uri",
				pid: "ownerA:repoA:non-existing",
			},
			[]string{"ownerA:repoA:pipeline1", "ownerA:repoA:pipeline2"},
			&model.Trigger{
				Event: "event:test:uri", Secret: "secretA", Pipelines: []model.Pipeline{
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline1"},
					{RepoOwner: "ownerA", RepoName: "repoA", Name: "pipeline2"},
				},
			},
			&storeMock{},
			expected{0, 2},
			true,
		},
		{
			"try to delete pipeline from non-existing trigger",
			fields{redisPool: &RedisPoolMock{}},
			args{
				id:  "event:test:uri",
				pid: "ownerA:repoA:pipeline1",
			},
			[]string{},
			nil,
			&storeMock{},
			expected{0, 0},
			true,
		},
		{
			"try to delete bad URI pipeline from trigger",
			fields{redisPool: &RedisPoolMock{}},
			args{
				id:  "event:test:uri",
				pid: "pipeline1",
			},
			[]string{},
			nil,
			&storeMock{},
			expected{0, 2},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:  tt.fields.redisPool,
				deleteFunc: tt.mock.Delete,
			}
			// remove command
			r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", getTriggerKey(tt.args.id), tt.args.pid).Expect(tt.expected.remove)
			// get remaining
			r.redisPool.GetConn().(*redigomock.Conn).Command("ZCARD", getTriggerKey(tt.args.id)).Expect(tt.expected.remain)
			// if removed and no remaining -> delete trigger
			if tt.expected.remove > 0 && tt.expected.remain == 0 {
				tt.mock.On("Delete", tt.args.id).Return(nil)
			}
			// perform delete
			if err := r.DeletePipeline(tt.args.id, tt.args.pid); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.DeletePipeline() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert expectations
			tt.mock.AssertExpectations(t)
		})
	}
}
