package backend

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/provider"
	"github.com/codefresh-io/hermes/pkg/util"
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

func TestMain(m *testing.M) {
	util.TestMode = true
	os.Exit(m.Run())
}

func TestRedisStore_Ping(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			"happy ping",
			"PONG",
			false,
		},
		{
			"failed ping - no Redis",
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
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

func TestRedisStore_GetPipelinesForTriggers(t *testing.T) {
	type args struct {
		id        string
		pipelines []string
	}
	tests := []struct {
		name         string
		args         []args
		expected     []string
		wantRedisErr error
		wantEmptyErr bool
	}{
		{
			"get single trigger pipelines",
			[]args{
				{
					id:        "test:uri",
					pipelines: []string{"puid-1", "puid-2", "puid-3"},
				},
			},
			[]string{"puid-1", "puid-2", "puid-3"},
			nil,
			false,
		},
		{
			"get multi trigger pipelines",
			[]args{
				{
					id:        "test-1:uri",
					pipelines: []string{"puid-1", "puid-2", "puid-3"},
				},
				{
					id:        "test-2:uri",
					pipelines: []string{"puid-4", "puid-5", "puid-6"},
				},
			},
			[]string{"puid-1", "puid-2", "puid-3", "puid-4", "puid-5", "puid-6"},
			nil,
			false,
		},
		{
			"get multi trigger pipelines (duplicate)",
			[]args{
				{
					id:        "test-1:uri",
					pipelines: []string{"puid-1", "puid-2", "puid-3"},
				},
				{
					id:        "test-2:uri",
					pipelines: []string{"puid-2", "puid-3", "puid-4"},
				},
			},
			[]string{"puid-1", "puid-2", "puid-3", "puid-4"},
			nil,
			false,
		},
		{
			"get all pipelines",
			[]args{
				{
					id:        "",
					pipelines: []string{"puid-1", "puid-2", "puid-3", "puid-4", "puid-5", "puid-6"},
				},
			},
			[]string{"puid-1", "puid-2", "puid-3", "puid-4", "puid-5", "puid-6"},
			nil,
			false,
		},
		{
			"get all pipelines with redis error",
			[]args{
				{
					id:        "",
					pipelines: []string{"puid-1", "puid-2", "puid-3", "puid-4", "puid-5", "puid-6"},
				},
			},
			nil,
			errors.New("REDIS error"),
			false,
		},
		{
			"get all pipelines with redis ErrNil",
			[]args{
				{
					id:        "",
					pipelines: []string{"puid-1", "puid-2", "puid-3", "puid-4", "puid-5", "puid-6"},
				},
			},
			nil,
			redis.ErrNil,
			false,
		},
		{
			"get all pipelines with empty result",
			[]args{
				{
					id:        "",
					pipelines: []string{},
				},
			},
			[]string{},
			nil,
			true,
		},
		{
			"get trigger pipelines ZRANGE error",
			[]args{
				{
					id:        "test:uri",
					pipelines: nil,
				},
			},
			nil,
			errors.New("REDIS error"),
			true,
		},
		{
			"get trigger pipelines ZRANGE error",
			[]args{
				{
					id:        "test:uri",
					pipelines: nil,
				},
			},
			nil,
			redis.ErrNil,
			true,
		},
		{
			"get trigger pipelines EMPTY error",
			[]args{
				{
					id:        "test:uri",
					pipelines: nil,
				},
			},
			nil,
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
			}
			for _, _arg := range tt.args {
				if _arg.id == "" {
					cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("KEYS", getPipelineKey(_arg.id))
					if tt.wantRedisErr != nil {
						cmd.ExpectError(tt.wantRedisErr)
					} else {
						cmd.Expect(util.InterfaceSlice(_arg.pipelines))
					}
				} else {
					cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(_arg.id), 0, -1)
					if tt.wantRedisErr != nil {
						cmd.ExpectError(tt.wantRedisErr)
					} else {
						cmd.Expect(util.InterfaceSlice(_arg.pipelines))
					}
				}
			}
			var ids []string
			for _, _arg := range tt.args {
				if _arg.id != "" {
					ids = append(ids, _arg.id)
				}
			}
			got, err := r.GetPipelinesForTriggers(ids)
			if (err != nil) != (tt.wantRedisErr != nil || tt.wantEmptyErr) {
				t.Errorf("RedisStore.GetPipelinesForTriggers() error = %v, wantErr %v", err, (tt.wantRedisErr != nil || tt.wantEmptyErr))
				return
			}
			if (len(got) > 0 || len(tt.expected) > 0) && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("RedisStore.GetPipelinesForTriggers() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRedisStore_GetSecret(t *testing.T) {
	type args struct {
		eventURI string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "get secret",
			args:    args{eventURI: "test"},
			want:    "123456789",
			wantErr: false,
		},
		{
			name:    "get secret missing",
			args:    args{eventURI: "test"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
			}
			if tt.wantErr {
				r.redisPool.GetConn().(*redigomock.Conn).Command("HGET", getEventKey(tt.args.eventURI), "secret").ExpectError(fmt.Errorf("HGET error"))
			} else {
				r.redisPool.GetConn().(*redigomock.Conn).Command("HGET", getEventKey(tt.args.eventURI), "secret").Expect(tt.want)
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
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"create trigger for non-existing pipeline",
			args{
				pipeline: "non-existing-pipeline",
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			true,
			redisErrors{false, false, false, false},
		},
		{
			"fail start transaction",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			true,
			redisErrors{true, false, false, false},
		},
		{
			"fail adding pipeline to Triggers map",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			true,
			redisErrors{false, true, false, false},
		},
		{
			"fail adding events to Pipelines map",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			true,
			redisErrors{false, false, true, false},
		},
		{
			"fail exec transaction",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test:1", "uri:test:2"},
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
			var params []interface{}

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
			params = []interface{}{getPipelineKey(tt.args.pipeline), 0}
			params = append(params, util.InterfaceSlice(tt.args.events)...)
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", params...)
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
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"delete single event trigger for pipeline",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"fail start transaction",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			true,
			redisErrors{true, false, false, false},
		},
		{
			"fail remove pipeline from Triggers map",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			true,
			redisErrors{false, true, false, false},
		},
		{
			"fail remove events from Pipelines map",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test:1", "uri:test:2"},
			},
			true,
			redisErrors{false, false, true, false},
		},
		{
			"fail exec transaction",
			args{
				pipeline: "owner:repo:test",
				events:   []string{"uri:test:1", "uri:test:2"},
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
			var params []interface{}
			cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI")
			if tt.errs.multi {
				cmd.ExpectError(errors.New("MULTI error"))
				goto Invoke
			} else {
				cmd.Expect("OK!")
			}
			// remove pipeline from Triggers
			for _, event := range tt.args.events {
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", getTriggerKey(event), tt.args.pipeline)
				if tt.errs.zrem1 {
					cmd.ExpectError(errors.New("ZREM error"))
					goto EndTransaction
				}
			}
			// remove events from Pipelines
			params = []interface{}{getPipelineKey(tt.args.pipeline)}
			params = append(params, util.InterfaceSlice(tt.args.events)...)
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", params...)
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

func TestRedisStore_DeleteTriggersForEvent(t *testing.T) {
	type redisErrors struct {
		multi bool
		zrem1 bool
		zrem2 bool
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
			"delete triggers for event",
			args{
				event:     "uri",
				pipelines: []string{"p1", "p2", "p3"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"delete single pipeline trigger",
			args{
				event:     "uri",
				pipelines: []string{"pipeline"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"fail start transaction",
			args{
				event:     "uri",
				pipelines: []string{"p1", "p2", "p3"},
			},
			true,
			redisErrors{true, false, false, false},
		},
		{
			"fail remove pipeline from Pipelines",
			args{
				event:     "uri",
				pipelines: []string{"p1", "p2", "p3"},
			},
			true,
			redisErrors{false, true, false, false},
		},
		{
			"fail remove events from Triggers",
			args{
				event:     "uri",
				pipelines: []string{"p1", "p2", "p3"},
			},
			true,
			redisErrors{false, false, true, false},
		},
		{
			"fail exec transaction",
			args{
				event:     "uri",
				pipelines: []string{"p1", "p2", "p3"},
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
			var params []interface{}
			cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI")
			if tt.errs.multi {
				cmd.ExpectError(errors.New("MULTI error"))
				goto Invoke
			} else {
				cmd.Expect("OK!")
			}
			// remove event from Pipelines
			for _, pipeline := range tt.args.pipelines {
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", getPipelineKey(pipeline), tt.args.event)
				if tt.errs.zrem1 {
					cmd.ExpectError(errors.New("ZREM error"))
					goto EndTransaction
				}
			}
			// remove pipelines from Triggers
			params = []interface{}{getTriggerKey(tt.args.event)}
			params = append(params, util.InterfaceSlice(tt.args.pipelines)...)
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", params...)
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
			if err := r.DeleteTriggersForEvent(tt.args.event, tt.args.pipelines); (err != nil) != tt.wantErr {
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
				event:     "uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			false,
			redisErrors{false, false, false, false},
		},
		{
			"create event trigger for non-existing pipeline",
			args{
				event:     "uri:test",
				pipelines: []string{"non-existing-pipeline", "owner:repo:test"},
			},
			true,
			redisErrors{false, false, false, false},
		},
		{
			"fail start transaction",
			args{
				event:     "uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			true,
			redisErrors{true, false, false, false},
		},
		{
			"fail adding pipeline to Triggers map",
			args{
				event:     "uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			true,
			redisErrors{false, true, false, false},
		},
		{
			"fail adding events to Pipelines map",
			args{
				event:     "uri:test",
				pipelines: []string{"owner:repo:test:1", "owner:repo:test:2"},
			},
			true,
			redisErrors{false, false, true, false},
		},
		{
			"fail exec transaction",
			args{
				event:     "uri:test",
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
			var params []interface{}
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
			params = []interface{}{getTriggerKey(tt.args.event), 0}
			params = append(params, util.InterfaceSlice(tt.args.pipelines)...)
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZADD", params...)
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

func TestRedisStore_ListTriggersForEvents(t *testing.T) {
	type redisErrors struct {
		keys   bool
		zrange bool
	}
	type args struct {
		events []string
	}
	type triggers struct {
		event     string
		pipelines []string
	}
	tests := []struct {
		name     string
		args     args
		triggers []triggers
		want     []model.Trigger
		errs     redisErrors
		wantErr  bool
	}{
		{
			name: "list triggers for single event",
			args: args{
				events: []string{"uri:test"},
			},
			triggers: []triggers{
				triggers{
					event:     "uri:test",
					pipelines: []string{"pipeline-1", "pipeline-2"},
				},
			},
			want: []model.Trigger{
				model.Trigger{Event: "uri:test", Pipeline: "pipeline-1"},
				model.Trigger{Event: "uri:test", Pipeline: "pipeline-2"},
			},
			errs: redisErrors{false, false},
		},
		{
			name: "list triggers for multiple event",
			args: args{
				events: []string{"uri:test:1", "uri:test:2"},
			},
			triggers: []triggers{
				triggers{
					event:     "uri:test:1",
					pipelines: []string{"pipeline-1", "pipeline-2"},
				},
				triggers{
					event:     "uri:test:2",
					pipelines: []string{"pipeline-2", "pipeline-3"},
				},
			},
			want: []model.Trigger{
				model.Trigger{Event: "uri:test:1", Pipeline: "pipeline-1"},
				model.Trigger{Event: "uri:test:1", Pipeline: "pipeline-2"},
				model.Trigger{Event: "uri:test:2", Pipeline: "pipeline-2"},
				model.Trigger{Event: "uri:test:2", Pipeline: "pipeline-3"},
			},
			errs: redisErrors{false, false},
		},
		{
			name: "fail to find trigger by event",
			args: args{
				events: []string{"non-existing-event"},
			},
			errs:    redisErrors{true, false},
			wantErr: true,
		},
		{
			name: "fail to find pipelines for event",
			args: args{
				events: []string{"uri:test"},
			},
			triggers: []triggers{
				triggers{
					event: "uri:test",
				},
				triggers{
					event:     "uri:test:other",
					pipelines: []string{"pipeline-2", "pipeline-3"},
				},
			},
			errs:    redisErrors{false, true},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
			}
			// get keys from Triggers Set
			keys := make([]string, 0)
			for _, event := range tt.args.events {
				cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("KEYS", getTriggerKey(event))
				if tt.errs.keys {
					cmd.ExpectError(errors.New("KEYS error"))
					goto Invoke
				} else {
					for _, t := range tt.triggers {
						if event == t.event {
							keys = util.MergeStrings(keys, []string{getTriggerKey(t.event)})
						}
					}
					cmd.Expect(util.InterfaceSlice(keys))
				}
			}
			// get pipelines from Triggers Set
			for _, k := range keys {
				cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", k, 0, -1)
				if tt.errs.zrange {
					cmd.ExpectError(errors.New("ZRANGE error"))
					goto Invoke
				} else {
					for _, t := range tt.triggers {
						// select pipelines for key
						if getTriggerKey(t.event) == k {
							cmd.Expect(util.InterfaceSlice(t.pipelines))
							break
						}
					}
				}
			}
		Invoke:
			got, err := r.ListTriggersForEvents(tt.args.events)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.ListTriggersForEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.ListTriggersForEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_ListTriggersForPipelines(t *testing.T) {
	type redisErrors struct {
		keys   bool
		zrange bool
	}
	type pipelines struct {
		pipeline string
		events   []string
	}
	type args struct {
		pipelines []string
	}
	tests := []struct {
		name      string
		args      args
		pipelines []pipelines
		want      []model.Trigger
		errs      redisErrors
		wantErr   bool
	}{
		{
			name: "list triggers for single pipeline",
			args: args{
				pipelines: []string{"test-pipeline"},
			},
			pipelines: []pipelines{
				pipelines{
					pipeline: "test-pipeline",
					events:   []string{"event-1", "event-2"},
				},
			},
			want: []model.Trigger{
				model.Trigger{Event: "event-1", Pipeline: "test-pipeline"},
				model.Trigger{Event: "event-2", Pipeline: "test-pipeline"},
			},
			errs:    redisErrors{false, false},
			wantErr: false,
		},
		{
			name: "list triggers for multiple pipeline",
			args: args{
				pipelines: []string{"test-pipeline-1", "test-pipeline-2"},
			},
			pipelines: []pipelines{
				pipelines{
					pipeline: "test-pipeline-1",
					events:   []string{"event-1", "event-2"},
				},
				pipelines{
					pipeline: "test-pipeline-2",
					events:   []string{"event-2", "event-3"},
				},
			},
			want: []model.Trigger{
				model.Trigger{Event: "event-1", Pipeline: "test-pipeline-1"},
				model.Trigger{Event: "event-2", Pipeline: "test-pipeline-1"},
				model.Trigger{Event: "event-2", Pipeline: "test-pipeline-2"},
				model.Trigger{Event: "event-3", Pipeline: "test-pipeline-2"},
			},
			errs:    redisErrors{false, false},
			wantErr: false,
		},
		{
			name: "fail to find triggers by pipeline",
			args: args{
				pipelines: []string{"non-existing-pipeline"},
			},
			pipelines: nil,
			want:      nil,
			errs:      redisErrors{true, false},
			wantErr:   true,
		},
		{
			name: "fail to find events for pipeline",
			args: args{
				pipelines: []string{"test-pipeline"},
			},
			pipelines: []pipelines{
				pipelines{
					pipeline: "test-pipeline",
					events:   nil,
				},
				pipelines{
					pipeline: "test-pipeline-2",
					events:   []string{"event-1", "event-2"},
				},
			},
			want:    nil,
			errs:    redisErrors{false, true},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
			}
			// get keys from Pipelines Set
			keys := make([]string, 0)
			for _, pipeline := range tt.args.pipelines {
				cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("KEYS", getPipelineKey(pipeline))
				if tt.errs.keys {
					cmd.ExpectError(errors.New("KEYS error"))
					goto Invoke
				} else {
					for _, t := range tt.pipelines {
						if pipeline == t.pipeline {
							keys = util.MergeStrings(keys, []string{getPipelineKey(t.pipeline)})
						}
					}
					cmd.Expect(util.InterfaceSlice(keys))
				}
			}
			// get events from Pipeline Set
			for _, k := range keys {
				cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", k, 0, -1)
				if tt.errs.zrange {
					cmd.ExpectError(errors.New("ZRANGE error"))
					goto Invoke
				} else {
					for _, t := range tt.pipelines {
						// select pipelines for key
						if getPipelineKey(t.pipeline) == k {
							cmd.Expect(util.InterfaceSlice(t.events))
							break
						}
					}
				}
			}
		Invoke:
			got, err := r.ListTriggersForPipelines(tt.args.pipelines)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.ListTriggersForPipelines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.ListTriggersForPipelines() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_GetEvent(t *testing.T) {
	type args struct {
		event string
	}
	type expect struct {
		fields map[string]string
	}
	tests := []struct {
		name    string
		args    args
		expect  expect
		want    *model.Event
		wantErr error
		keyErr  bool
	}{
		{
			name: "get existing event",
			args: args{event: "uri:test"},
			expect: expect{
				fields: map[string]string{
					"type":        "test-type",
					"kind":        "test-kind",
					"secret":      "test-secret",
					"endpoint":    "http://endpoint",
					"description": "test-desc",
					"status":      "test",
					"help":        "test-help",
				},
			},
			want: &model.Event{
				URI:    "uri:test",
				Type:   "test-type",
				Kind:   "test-kind",
				Secret: "test-secret",
				EventInfo: model.EventInfo{
					Endpoint:    "http://endpoint",
					Description: "test-desc",
					Status:      "test",
					Help:        "test-help",
				},
			},
		},
		{
			name:    "get non-existing event",
			args:    args{event: "non-existing:event:uri:test"},
			expect:  expect{},
			wantErr: model.ErrEventNotFound,
		},
		{
			name:    "get non-existing event REDIS error",
			args:    args{event: "non-existing:event:uri:test"},
			expect:  expect{},
			wantErr: errors.New("HGETALL error"),
		},
		{
			name:    "try getting event with invalid key",
			args:    args{event: "uri:*"},
			expect:  expect{},
			wantErr: model.ErrNotSingleKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
			}
			cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("HGETALL", getEventKey(tt.args.event))
			if tt.wantErr != nil && tt.wantErr != model.ErrEventNotFound {
				cmd.ExpectError(tt.wantErr)
			} else {
				cmd.ExpectMap(tt.expect.fields)
			}
			got, err := r.GetEvent(tt.args.event)
			if err != tt.wantErr {
				t.Errorf("RedisStore.GetEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.GetEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_GetEvents(t *testing.T) {
	type expect struct {
		keys   []string
		fields []map[string]string
	}
	type args struct {
		eventType string
		kind      string
		filter    string
	}
	tests := []struct {
		name    string
		args    args
		expect  expect
		want    []model.Event
		wantErr bool
	}{
		{
			name: "get all trigger events",
			args: args{},
			expect: expect{
				keys: []string{"uri:1", "uri:2", "uri:3"},
				fields: []map[string]string{
					{"type": "t1", "kind": "k1", "secret": "s1"},
					{"type": "t2", "kind": "k2", "secret": "s2"},
					{"type": "t3", "kind": "k3", "secret": "s3"},
				},
			},
			want: []model.Event{
				{URI: "uri:1", Type: "t1", Kind: "k1", Secret: "s1"},
				{URI: "uri:2", Type: "t2", Kind: "k2", Secret: "s2"},
				{URI: "uri:3", Type: "t3", Kind: "k3", Secret: "s3"},
			},
		},
		{
			name: "get trigger events by type",
			args: args{eventType: "T"},
			expect: expect{
				keys: []string{"uri:1", "uri:2", "uri:3"},
				fields: []map[string]string{
					{"type": "t1", "kind": "k1", "secret": "s1"},
					{"type": "T", "kind": "k2", "secret": "s2"},
					{"type": "T", "kind": "k3", "secret": "s3"},
				},
			},
			want: []model.Event{
				{URI: "uri:2", Type: "T", Kind: "k2", Secret: "s2"},
				{URI: "uri:3", Type: "T", Kind: "k3", Secret: "s3"},
			},
		},
		{
			name: "get trigger events by filter",
			args: args{filter: "uri:2*"},
			expect: expect{
				keys: []string{"uri:21", "uri:22"},
				fields: []map[string]string{
					{"type": "t2", "kind": "k2", "secret": "s2"},
					{"type": "t3", "kind": "k3", "secret": "s3"},
				},
			},
			want: []model.Event{
				{URI: "uri:21", Type: "t2", Kind: "k2", Secret: "s2"},
				{URI: "uri:22", Type: "t3", Kind: "k3", Secret: "s3"},
			},
		},
		{
			name: "get trigger events by type and kind",
			args: args{eventType: "T", kind: "K"},
			expect: expect{
				keys: []string{"uri:1", "uri:2", "uri:3"},
				fields: []map[string]string{
					{"type": "t1", "kind": "k1", "secret": "s1"},
					{"type": "T", "kind": "K", "secret": "s2"},
					{"type": "T", "kind": "k3", "secret": "s3"},
				},
			},
			want: []model.Event{
				{URI: "uri:2", Type: "T", Kind: "K", Secret: "s2"},
			},
		},
		{
			name: "get no trigger events by type and kind",
			args: args{eventType: "T", kind: "K"},
			expect: expect{
				keys: []string{"uri:1", "uri:2", "uri:3"},
				fields: []map[string]string{
					{"type": "t1", "kind": "k1", "secret": "s1"},
					{"type": "t2", "kind": "k2", "secret": "s2"},
					{"type": "t3", "kind": "k3", "secret": "s3"},
				},
			},
		},
		{
			name:    "keys error",
			args:    args{},
			expect:  expect{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
			}
			// mock getting trigger event keys
			cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("KEYS", getEventKey(tt.args.filter))
			if tt.wantErr {
				cmd.ExpectError(errors.New("KEYS error"))
				goto Invoke
			} else {
				cmd.Expect(util.InterfaceSlice(tt.expect.keys))
			}
			// mock scanning trough all trigger events
			for i, k := range tt.expect.keys {
				cmd := r.redisPool.GetConn().(*redigomock.Conn).Command("HGETALL", getEventKey(k))
				cmd.ExpectMap(tt.expect.fields[i])
			}

			// invoke
		Invoke:
			got, err := r.GetEvents(tt.args.eventType, tt.args.kind, tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.GetEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (len(got) != 0 || len(tt.want) != 0) && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.GetEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisStore_DeleteEvent(t *testing.T) {
	type redisErrors struct {
		multi      bool
		zrange     bool
		delEvent   bool
		delTrigger bool
		zrem       bool
		exec       bool
	}
	type expected struct {
		pipelines []string
	}
	type args struct {
		event   string
		context string
	}
	tests := []struct {
		name     string
		args     args
		expected expected
		errs     redisErrors
		wantErr  error
	}{
		{
			name: "delete existing trigger event",
			args: args{event: "uri:test"},
			expected: expected{
				pipelines: []string{"p1", "p2", "p3"},
			},
		},
		{
			name:    "try deleting event with invalid key",
			args:    args{event: "uri:*"},
			wantErr: model.ErrNotSingleKey,
		},
		{
			name:    "zrange error",
			args:    args{event: "uri:test"},
			wantErr: errors.New("REDIS error"),
			errs:    redisErrors{zrange: true},
		},
		{
			name: "multi error",
			args: args{event: "uri:test"},
			expected: expected{
				pipelines: []string{"p1", "p2", "p3"},
			},
			wantErr: errors.New("REDIS error"),
			errs:    redisErrors{multi: true},
		},
		{
			name: "del event error",
			args: args{event: "uri:test"},
			expected: expected{
				pipelines: []string{"p1", "p2", "p3"},
			},
			wantErr: errors.New("REDIS error"),
			errs:    redisErrors{delEvent: true},
		},
		{
			name: "del trigger error",
			args: args{event: "uri:test"},
			expected: expected{
				pipelines: []string{"p1", "p2", "p3"},
			},
			wantErr: errors.New("REDIS error"),
			errs:    redisErrors{delTrigger: true},
		},
		{
			name: "zrem trigger error",
			args: args{event: "uri:test"},
			expected: expected{
				pipelines: []string{"p1", "p2", "p3"},
			},
			wantErr: errors.New("REDIS error"),
			errs:    redisErrors{zrem: true},
		},
		{
			name: "exec error",
			args: args{event: "uri:test"},
			expected: expected{
				pipelines: []string{"p1", "p2", "p3"},
			},
			wantErr: errors.New("REDIS error"),
			errs:    redisErrors{exec: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedisStore{
				redisPool: &RedisPoolMock{},
			}
			// mock Redis
			// expect Redis transaction open
			var cmd *redigomock.Cmd
			// get trigger event pipelines
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZRANGE", getTriggerKey(tt.args.event), 0, -1)
			if tt.errs.zrange {
				cmd.ExpectError(tt.wantErr)
				goto Invoke
			} else {
				cmd.Expect(util.InterfaceSlice(tt.expected.pipelines))
			}
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI")
			if tt.errs.multi {
				cmd.ExpectError(tt.wantErr)
				goto Invoke
			} else {
				cmd.Expect("OK!")
			}
			// delete event
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", getEventKey(tt.args.event))
			if tt.errs.delEvent {
				cmd.ExpectError(tt.wantErr)
				goto EndTransaction
			} else {
				cmd.Expect("QUEUED")
			}
			// delete trigger
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("DEL", getTriggerKey(tt.args.event))
			if tt.errs.delTrigger {
				cmd.ExpectError(tt.wantErr)
				goto EndTransaction
			} else {
				cmd.Expect("QUEUED")
			}
			// delete trigger
			for _, pipeline := range tt.expected.pipelines {
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("ZREM", getPipelineKey(pipeline), tt.args.event)
				if tt.errs.zrem {
					cmd.ExpectError(tt.wantErr)
					goto EndTransaction
				} else {
					cmd.Expect("QUEUED")
				}
			}

		EndTransaction:
			// discard transaction on error
			if (tt.errs.delEvent || tt.errs.delTrigger || tt.errs.zrem) && !tt.errs.exec {
				// expect transaction discard on error
				r.redisPool.GetConn().(*redigomock.Conn).Command("DISCARD").Expect("OK!")
			} else {
				// expect Redis transaction exec
				cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("EXEC")
				if tt.errs.exec {
					cmd.ExpectError(tt.wantErr)
				} else {
					cmd.Expect("OK!")
				}
			}

		Invoke:
			if err := r.DeleteEvent(tt.args.event, tt.args.context); err != tt.wantErr {
				t.Errorf("RedisStore.DeleteEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisStore_CreateEvent(t *testing.T) {
	type redisErrors struct {
		multi          bool
		hsetnxType     bool
		hsetnxKind     bool
		hsetnxSecret   bool
		hsetnxDesc     bool
		hsetnxEndpoint bool
		hsetnxHelp     bool
		hsetnxStatus   bool
		exec           bool
	}
	type eventErrors struct {
		uri       error
		subscribe error
		info      error
	}
	type expected struct {
		eventURI    string
		info        *model.EventInfo
		credentials map[string]string
	}
	type args struct {
		eventType string
		kind      string
		secret    string
		context   string
		values    map[string]string
	}
	tests := []struct {
		name         string
		args         args
		expected     expected
		want         *model.Event
		wantErr      bool
		wantEventErr eventErrors
		errs         redisErrors
	}{
		{
			name: "create event",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			want: &model.Event{
				URI:       "type:kind:test",
				Type:      "type",
				Kind:      "kind",
				Secret:    "XXX",
				EventInfo: model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"}},
			errs: redisErrors{},
		},
		{
			name:         "fail to construct URI",
			args:         args{eventType: "type", kind: "kind", secret: "XXX"},
			expected:     expected{eventURI: ""},
			wantEventErr: eventErrors{uri: errors.New("URI error")},
			errs:         redisErrors{},
		},
		{
			name:         "fail to subscribe to event",
			args:         args{eventType: "type", kind: "kind", secret: "XXX"},
			expected:     expected{eventURI: "type:kind:test"},
			wantEventErr: eventErrors{subscribe: errors.New("Subscribe error")},
			errs:         redisErrors{},
		},
		{
			name: "fail to subscribe to event (not implemented)",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			want: &model.Event{
				URI:       "type:kind:test",
				Type:      "type",
				Kind:      "kind",
				Secret:    "XXX",
				EventInfo: model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"}},
			wantEventErr: eventErrors{subscribe: provider.ErrNotImplemented},
			errs:         redisErrors{},
		},
		{
			name:         "fail to subscribe to event and get info fallback fails too",
			args:         args{eventType: "type", kind: "kind", secret: "XXX"},
			expected:     expected{eventURI: "type:kind:test"},
			wantEventErr: eventErrors{subscribe: provider.ErrNotImplemented, info: errors.New("Info error")},
			errs:         redisErrors{},
		},
		{
			name: "fail start transaction",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{multi: true},
		},
		{
			name: "fail update type",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{hsetnxType: true},
		},
		{
			name: "fail update kind",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{hsetnxKind: true},
		},
		{
			name: "fail update secret",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{hsetnxSecret: true},
		},
		{
			name: "fail update description",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{hsetnxDesc: true},
		},
		{
			name: "fail update endpoint",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{hsetnxEndpoint: true},
		},
		{
			name: "fail update help",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{hsetnxHelp: true},
		},
		{
			name: "fail update status",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{hsetnxStatus: true},
		},
		{
			name: "fail exec transaction",
			args: args{eventType: "type", kind: "kind", secret: "XXX"},
			expected: expected{
				eventURI: "type:kind:test",
				info:     &model.EventInfo{Endpoint: "test-endpoint", Description: "test-desc", Help: "test-help", Status: "test-status"},
			},
			wantErr: true,
			errs:    redisErrors{exec: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd *redigomock.Cmd
			mock := provider.NewEventProviderMock()
			r := &RedisStore{
				redisPool:     &RedisPoolMock{},
				eventProvider: mock,
			}
			// mock EventProvider calls
			call := mock.On("ConstructEventURI", tt.args.eventType, tt.args.kind, tt.args.values)
			if tt.wantEventErr.uri != nil {
				call.Return("", tt.wantEventErr.uri)
				goto Invoke
			} else {
				call.Return(tt.expected.eventURI, nil)
			}
			call = mock.On("SubscribeToEvent", tt.expected.eventURI, tt.args.secret, tt.expected.credentials)
			if tt.wantEventErr.subscribe != nil {
				call.Return(nil, tt.wantEventErr.subscribe)
				if tt.wantEventErr.subscribe == provider.ErrNotImplemented {
					call = mock.On("GetEventInfo", tt.expected.eventURI, tt.args.secret)
					if tt.wantEventErr.info != nil {
						call.Return(nil, tt.wantEventErr.info)
						goto Invoke
					} else {
						call.Return(tt.expected.info, nil)
					}
				} else {
					goto Invoke
				}
			} else {
				call.Return(tt.expected.info, nil)
			}
			// mock Redis
			// expect Redis transaction open
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("MULTI")
			if tt.errs.multi {
				cmd.ExpectError(errors.New("MULTI error"))
				goto Invoke
			} else {
				cmd.Expect("OK!")
			}
			// store Event type
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("HSETNX", getEventKey(tt.expected.eventURI), "type", tt.args.eventType)
			if tt.errs.hsetnxType {
				cmd.ExpectError(errors.New("HSETNX error"))
				goto EndTransaction
			}
			// store Event kind
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("HSETNX", getEventKey(tt.expected.eventURI), "kind", tt.args.kind)
			if tt.errs.hsetnxKind {
				cmd.ExpectError(errors.New("HSETNX error"))
				goto EndTransaction
			}
			// store Event secret
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("HSETNX", getEventKey(tt.expected.eventURI), "secret", tt.args.secret)
			if tt.errs.hsetnxSecret {
				cmd.ExpectError(errors.New("HSETNX error"))
				goto EndTransaction
			}
			// store Event description
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("HSETNX", getEventKey(tt.expected.eventURI), "description", tt.expected.info.Description)
			if tt.errs.hsetnxDesc {
				cmd.ExpectError(errors.New("HSETNX error"))
				goto EndTransaction
			}
			// store Event endpoint
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("HSETNX", getEventKey(tt.expected.eventURI), "endpoint", tt.expected.info.Endpoint)
			if tt.errs.hsetnxEndpoint {
				cmd.ExpectError(errors.New("HSETNX error"))
				goto EndTransaction
			}
			// store Event help
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("HSETNX", getEventKey(tt.expected.eventURI), "help", tt.expected.info.Help)
			if tt.errs.hsetnxHelp {
				cmd.ExpectError(errors.New("HSETNX error"))
				goto EndTransaction
			}
			// store Event status
			cmd = r.redisPool.GetConn().(*redigomock.Conn).Command("HSETNX", getEventKey(tt.expected.eventURI), "status", tt.expected.info.Status)
			if tt.errs.hsetnxStatus {
				cmd.ExpectError(errors.New("HSETNX error"))
				goto EndTransaction
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
			// invoke method under test
			got, err := r.CreateEvent(tt.args.eventType, tt.args.kind, tt.args.secret, tt.args.context, tt.args.values)
			if (err != nil) != (tt.wantErr ||
				tt.wantEventErr.info != nil ||
				(tt.wantEventErr.subscribe != nil &&
					tt.wantEventErr.subscribe != provider.ErrNotImplemented) ||
				tt.wantEventErr.uri != nil) {
				t.Errorf("RedisStore.CreateEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RedisStore.CreateEvent() = %v, want %v", got, tt.want)
			}
			// assert mock
			mock.AssertExpectations(t)
		})
	}
}

func Test_checkSingleKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name: "valid key",
			key:  "registry:dockerhub:12345",
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "invalid key star",
			key:     "registry:*",
			wantErr: true,
		},
		{
			name:    "invalid key options",
			key:     "registry:[dockerhub,cfcr]",
			wantErr: true,
		},
		{
			name:    "invalid key q mark",
			key:     "registry:?hub",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkSingleKey(tt.key); (err != nil) != tt.wantErr {
				t.Errorf("checkSingleKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
