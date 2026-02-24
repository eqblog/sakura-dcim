package config

import "testing"

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "sakura",
		Password: "secret",
		DBName:   "sakura_dcim",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()
	expected := "postgres://sakura:secret@localhost:5432/sakura_dcim?sslmode=disable"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}

func TestRedisConfig_Addr(t *testing.T) {
	cfg := RedisConfig{Host: "redis", Port: 6379}
	addr := cfg.Addr()
	if addr != "redis:6379" {
		t.Errorf("Addr = %q, want %q", addr, "redis:6379")
	}
}

func TestDatabaseConfig_DSN_CustomPort(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "db.example.com",
		Port:     5433,
		User:     "admin",
		Password: "p@ss",
		DBName:   "mydb",
		SSLMode:  "require",
	}

	dsn := cfg.DSN()
	expected := "postgres://admin:p@ss@db.example.com:5433/mydb?sslmode=require"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}
