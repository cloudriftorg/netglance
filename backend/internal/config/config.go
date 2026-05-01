package config

import "os"

type Config struct {
	Bind    string
	DataDir string
}

func Load() Config {
	return Config{
		Bind:    envOr("NETGLANCE_BIND", ":8080"),
		DataDir: envOr("NETGLANCE_DATA_DIR", defaultDataDir()),
	}
}

func defaultDataDir() string {
	if _, err := os.Stat("/data"); err == nil {
		return "/data"
	}
	return "./data"
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
