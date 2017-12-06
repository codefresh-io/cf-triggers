package backend

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/garyburd/redigo/redis"
	"github.com/rafaeljusto/redigomock"
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

type CFMock struct{}

func (c *CFMock) RunPipeline(name string, repoOwner string, repoName string, vars map[string]string) error {
	return nil
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
		id      string
		want    model.Trigger
		wantErr bool
	}{
		{
			"get trigger by id",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			"test:1",
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
			"test:1",
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
			got, err := r.Get(tt.id)
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
		wantErr bool
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
			false,
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
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			if tt.wantErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("SET", tt.trigger.Event, tt.trigger.Secret).ExpectError(fmt.Errorf("SET error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("SET", tt.trigger.Event, tt.trigger.Secret).Expect("OK!")
				for _, p := range tt.trigger.Pipelines {
					jp, _ := json.Marshal(p)
					r.redisPool.GetConn().(*redigomock.Conn).Command("SADD", getTriggerKey(tt.trigger.Event), jp)
				}
			}
			if err := r.Add(tt.trigger); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
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
		id     string
		secret string
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
			args{id: "test", secret: "secretAAA"},
			"secretAAA",
			false,
		},
		{
			"check secret",
			fields{redisPool: &RedisPoolMock{}, pipelineSvc: &CFMock{}},
			args{id: "test", secret: "secretAAA"},
			"secretBBB",
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
			if err := r.CheckSecret(tt.args.id, tt.args.secret); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.CheckSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
