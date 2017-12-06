package backend

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/codefresh-io/cf-triggers/pkg/codefresh"
	"github.com/codefresh-io/cf-triggers/pkg/model"
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
		{"without prefix", "github.com:project:test", "trigger:github.com:project:test"},
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
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    model.Trigger
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			got, err := r.Get(tt.args.id)
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
	type args struct {
		trigger model.Trigger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			if err := r.Add(tt.args.trigger); (err != nil) != tt.wantErr {
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
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			if err := r.Delete(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisStore_Update(t *testing.T) {
	type fields struct {
		redisPool   RedisPoolService
		pipelineSvc codefresh.PipelineService
	}
	type args struct {
		t model.Trigger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			if err := r.Update(tt.args.t); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.Update() error = %v, wantErr %v", err, tt.wantErr)
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
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			if err := r.Run(tt.args.id, tt.args.vars); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.Run() error = %v, wantErr %v", err, tt.wantErr)
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
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool:   tt.fields.redisPool,
				pipelineSvc: tt.fields.pipelineSvc,
			}
			if err := r.CheckSecret(tt.args.id, tt.args.secret); (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.CheckSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
