package config

import (
	"flag"
	"os"
)

type Config struct {
	Addr         string
	ModelMode    string
	PromptPrefix string
	ServiceName  string
}

func Load() Config {
	cfg := Config{
		Addr:         ":8080",
		ModelMode:    "mock",
		PromptPrefix: "Demo:",
		ServiceName:  "tiny-llm",
	}

	flag.StringVar(&cfg.Addr, "addr", getenv("ADDR", cfg.Addr), "HTTP listen address")
	flag.StringVar(&cfg.ModelMode, "model-mode", getenv("MODEL_MODE", cfg.ModelMode), "model mode")
	flag.StringVar(&cfg.PromptPrefix, "prompt-prefix", getenv("PROMPT_PREFIX", cfg.PromptPrefix), "response prefix")
	flag.StringVar(&cfg.ServiceName, "service-name", getenv("SERVICE_NAME", cfg.ServiceName), "service name")
	flag.Parse()

	return cfg
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
