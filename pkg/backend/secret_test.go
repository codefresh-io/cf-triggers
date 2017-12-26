package backend

import (
	"testing"
)

func TestSecretChecker_Validate(t *testing.T) {
	type args struct {
		message string
		secret  string
		key     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"check secret",
			args{message: "hello world", secret: "secret", key: "secret"},
			false,
		},
		{
			"check secret error",
			args{message: "hello world", secret: "secret", key: "different"},
			true,
		},
		{
			"check secret signature sha1",
			args{message: "hello world", secret: "c61fe17e43c57ac8b18a1cb7b2e9ff666f506fa5", key: "secretKey"},
			false,
		},
		{
			"check empty secret",
			args{message: "hello world", secret: "", key: ""},
			false,
		},
		{
			"check no secret passed",
			args{message: "hello world", secret: "", key: "secretKey"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SecretChecker{}
			if err := s.Validate(tt.args.message, tt.args.secret, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("SecretChecker.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
