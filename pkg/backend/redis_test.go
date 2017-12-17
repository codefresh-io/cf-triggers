package backend

import (
	"encoding/json"
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

type CFMock struct {
	mock.Mock
}

func (c *CFMock) CheckPipelineExist(name string, repoOwner string, repoName string) error {
	args := c.Called(name, repoOwner, repoName)
	return args.Error(0)
}

func (c *CFMock) RunPipeline(name string, repoOwner string, repoName string, vars map[string]string) (string, error) {
	args := c.Called(name, repoOwner, repoName, vars)
	return args.String(0), args.Error(1)
}

func (c *CFMock) Ping() error {
	args := c.Called()
	return args.Error(0)
}

// helper function to convert []string to []interface{}
// see https://github.com/golang/go/wiki/InterfaceSlice
func interfaceSlice(slice []string, bytes bool) []interface{} {
	islice := make([]interface{}, len(slice))
	for i, v := range slice {
		if bytes {
			islice[i] = []uint8(v)
		} else {
			islice[i] = v
		}
	}
	return islice
}

func interfaceSlicePipelines(pipelines []model.Pipeline) []interface{} {
	islice := make([]interface{}, len(pipelines))
	for i, v := range pipelines {
		islice[i], _ = json.Marshal(v)
	}
	return islice
}

func Test_getTriggerKey(t *testing.T) {
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
		redisPool   RedisPoolService
		pipelineSvc codefresh.PipelineService
	}
	tests := []struct {
		name    string
		fields  fields
		filter  string
		keys    []string
		want    []model.Trigger
		wantErr bool
	}{
		{
			"get empty list",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			"",
			[]string{},
			[]model.Trigger{},
			false,
		},
		{
			"get all",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			"*",
			[]string{"test:1", "test:2"},
			[]model.Trigger{
				{Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "test", RepoOwner: "ownerA", RepoName: "repoA"},
				}},
				{Event: "test:2", Secret: "secretB", Pipelines: []model.Pipeline{
					{Name: "test", RepoOwner: "ownerB", RepoName: "repoB"},
				}},
			},
			false,
		},
		{
			"get one",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			"test:*",
			[]string{"test:1"},
			[]model.Trigger{
				{Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "test", RepoOwner: "ownerA", RepoName: "repoA"},
				}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			r.redisPool.GetConn().(*redigomock.Conn).Command("KEYS", getTriggerKey(tt.filter)).Expect(interfaceSlice(tt.keys, true))
			for i, k := range tt.keys {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", k).Expect(tt.want[i].Secret)
				r.redisPool.GetConn().(*redigomock.Conn).Command("SMEMBERS", getTriggerKey(k)).Expect(interfaceSlicePipelines(tt.want[i].Pipelines))
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

func TestRedisStore_Get(t *testing.T) {
	type fields struct {
		redisPool   RedisPoolService
		pipelineSvc codefresh.PipelineService
	}
	tests := []struct {
		name    string
		fields  fields
		want    model.Trigger
		wantErr bool
	}{
		{
			"get trigger by id",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "test", RepoOwner: "ownerA", RepoName: "repoA"},
				},
			},
			false,
		},
		{
			"get trigger GET error",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			model.Trigger{
				Event: "test:1", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "test", RepoOwner: "ownerA", RepoName: "repoA"},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			if tt.wantErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", tt.want.Event).ExpectError(fmt.Errorf("GET error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("GET", tt.want.Event).Expect(tt.want.Secret)
				r.redisPool.GetConn().(*redigomock.Conn).Command("SMEMBERS", getTriggerKey(tt.want.Event)).Expect(interfaceSlicePipelines(tt.want.Pipelines))
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

func TestRedisStore_Add(t *testing.T) {
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
			"add trigger",
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
			"add trigger with auto-generated secret",
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
			"add trigger SET error",
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
			"add trigger non-existing pipeline",
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
			"add trigger SADD error",
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
			// error cases
			if tt.wantErr[0] {
				r.redisPool.GetConn().(*redigomock.Conn).Command("SET", tt.trigger.Event, tt.trigger.Secret).ExpectError(fmt.Errorf("SET error"))
			} else {
				if tt.trigger.Secret == model.GenerateKeyword {
					r.redisPool.GetConn().(*redigomock.Conn).Command("SET", tt.trigger.Event, util.TestRandomString).Expect("OK!")
				} else {
					r.redisPool.GetConn().(*redigomock.Conn).Command("SET", tt.trigger.Event, tt.trigger.Secret).Expect("OK!")
				}
				for _, p := range tt.trigger.Pipelines {
					if tt.wantErr[1] {
						mock.On("CheckPipelineExist", p.Name, p.RepoOwner, p.RepoName).Return(codefresh.ErrPipelineNotFound)
						break
					} else {
						mock.On("CheckPipelineExist", p.Name, p.RepoOwner, p.RepoName).Return(nil)
					}
					jp, _ := json.Marshal(p)
					if tt.wantErr[2] {
						r.redisPool.GetConn().(*redigomock.Conn).Command("SADD", getTriggerKey(tt.trigger.Event), jp).ExpectError(fmt.Errorf("SADD error"))
						break
					} else {
						r.redisPool.GetConn().(*redigomock.Conn).Command("SADD", getTriggerKey(tt.trigger.Event), jp)
					}
				}
			}
			// perform function call
			if err := r.Add(tt.trigger); (err != nil) != (tt.wantErr[0] || tt.wantErr[1] || tt.wantErr[2]) {
				t.Errorf("RedisStore.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert expectation
			mock.AssertExpectations(t)
		})
	}
}

func TestRedisStore_Delete(t *testing.T) {
	type fields struct {
		redisPool   RedisPoolService
		pipelineSvc codefresh.PipelineService
	}
	type args struct {
		id string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantStrErr bool
		wantSetErr bool
	}{
		{
			"delete trigger",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test"},
			false,
			false,
		},
		{
			"delete trigger DEL STRING error",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test"},
			true,
			false,
		},
		{
			"delete trigger DEL SET error",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test"},
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
			if tt.wantStrErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", tt.args.id).ExpectError(fmt.Errorf("DEL STRING error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", tt.args.id).Expect("OK!")
			}
			if tt.wantSetErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", getTriggerKey(tt.args.id)).ExpectError(fmt.Errorf("DEL SET error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", getTriggerKey(tt.args.id)).Expect("OK!")
			}
			if err := r.Delete(tt.args.id); (err != nil) != (tt.wantStrErr || tt.wantSetErr) {
				t.Errorf("RedisStore.Delete() error = %v", err)
			}
		})
	}
}

func TestRedisStore_CheckSecret(t *testing.T) {
	type fields struct {
		redisPool   RedisPoolService
		pipelineSvc codefresh.PipelineService
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
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test", secret: "secretAAA", message: "hello world"},
			"secretAAA",
			false,
		},
		{
			"check secret error",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test", secret: "secretAAA", message: "hello world"},
			"secretBBB",
			true,
		},
		{
			"check secret signature sha1",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test", secret: "c61fe17e43c57ac8b18a1cb7b2e9ff666f506fa5", message: "hello world"},
			"secretKey",
			false,
		},
		{
			"check empty secret",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test", secret: "", message: "hello world"},
			"",
			false,
		},
		{
			"check no secret passed",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test", secret: "", message: "hello world"},
			"secretKey",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			r.redisPool.GetConn().(*redigomock.Conn).Command("GET", tt.args.id).Expect(tt.expectedSecret)
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
		id   string
		vars map[string]string
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
			args{id: "test:event", vars: map[string]string{"V1": "AAA", "V2": "BBB"}},
			model.Trigger{
				Event: "test:event", Secret: "secretA", Pipelines: []model.Pipeline{
					{Name: "pipeline1", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipeline2", RepoOwner: "ownerA", RepoName: "repoA"},
					{Name: "pipeline3", RepoOwner: "ownerA", RepoName: "repoB"},
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
				mock.On("RunPipeline", p.Name, p.RepoOwner, p.RepoName, tt.args.vars).Return(tt.want[i], nil)
			}
			// mock redis commands
			r.redisPool.GetConn().(*redigomock.Conn).Command("GET", tt.trigger.Event).Expect(tt.trigger.Secret)
			r.redisPool.GetConn().(*redigomock.Conn).Command("SMEMBERS", getTriggerKey(tt.trigger.Event)).Expect(interfaceSlicePipelines(tt.trigger.Pipelines))
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
