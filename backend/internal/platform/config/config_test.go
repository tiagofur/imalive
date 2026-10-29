package config

import (
	"log/slog"
	"strings"
	"testing"
)

func getenv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		version string
		want    Config
		wantErr string
	}{
		{
			name: "valid full configuration",
			env: map[string]string{
				"APP_ENV":      "production",
				"PORT":         "9000",
				"LOG_LEVEL":    "debug",
				"DATABASE_URL": "postgres://u:p@localhost:5432/imalive",
				"REDIS_URL":    "redis://localhost:6379/0",
			},
			version: "v0.1.0",
			want: Config{
				Env:         EnvProduction,
				Port:        "9000",
				LogLevel:    slog.LevelDebug,
				Version:     "v0.1.0",
				DatabaseURL: "postgres://u:p@localhost:5432/imalive",
				RedisURL:    "redis://localhost:6379/0",
			},
		},
		{
			name:    "defaults apply for optional variables",
			env:     map[string]string{"APP_ENV": "local"},
			version: "dev",
			want: Config{
				Env:      EnvLocal,
				Port:     DefaultPort,
				LogLevel: slog.LevelInfo,
				Version:  "dev",
			},
		},
		{
			name:    "missing APP_ENV fails fast",
			env:     map[string]string{},
			wantErr: "config: APP_ENV is required",
		},
		{
			name:    "unknown APP_ENV fails fast",
			env:     map[string]string{"APP_ENV": "prod"},
			wantErr: `config: APP_ENV "prod" is not one of`,
		},
		{
			name:    "invalid log level fails fast",
			env:     map[string]string{"APP_ENV": "local", "LOG_LEVEL": "verbose"},
			wantErr: `config: LOG_LEVEL "verbose" is not one of`,
		},
		{
			name:    "slog offset form INFO+2 fails fast (not in the documented enum)",
			env:     map[string]string{"APP_ENV": "local", "LOG_LEVEL": "INFO+2"},
			wantErr: `config: LOG_LEVEL "INFO+2" is not one of`,
		},
		{
			name:    "slog offset form ERROR+50 fails fast (would silence every record)",
			env:     map[string]string{"APP_ENV": "local", "LOG_LEVEL": "ERROR+50"},
			wantErr: `config: LOG_LEVEL "ERROR+50" is not one of`,
		},
		{
			name:    "uppercase INFO fails fast (enum is lowercase)",
			env:     map[string]string{"APP_ENV": "local", "LOG_LEVEL": "INFO"},
			wantErr: `config: LOG_LEVEL "INFO" is not one of`,
		},
		{
			name: "warn level parses",
			env:  map[string]string{"APP_ENV": "local", "LOG_LEVEL": "warn"},
			want: Config{Env: EnvLocal, Port: DefaultPort, LogLevel: slog.LevelWarn, Version: ""},
		},
		{
			name: "error level parses",
			env:  map[string]string{"APP_ENV": "local", "LOG_LEVEL": "error"},
			want: Config{Env: EnvLocal, Port: DefaultPort, LogLevel: slog.LevelError, Version: ""},
		},
		{
			name:    "non numeric port fails fast",
			env:     map[string]string{"APP_ENV": "local", "PORT": "http"},
			wantErr: `config: PORT "http" is not a valid TCP port`,
		},
		{
			name:    "port out of range fails fast",
			env:     map[string]string{"APP_ENV": "local", "PORT": "70000"},
			wantErr: `config: PORT "70000" is not a valid TCP port`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromEnv(getenv(tt.env), tt.version)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("FromEnv() = %+v, want error %q", got, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("FromEnv() error = %q, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("FromEnv() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("FromEnv() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
