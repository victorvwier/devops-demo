package config

import (
	"flag"
	"os"
)

type Config struct {
	Addr           string
	CatalogPath    string
	DefaultService string
	PromptPrefix   string
	FrontendTitle  string
}

func Load() Config {
	cfg := Config{
		Addr:           ":8080",
		CatalogPath:    getenv("CATALOG_PATH", "/etc/tiny-llm/catalog/services.json"),
		DefaultService: getenv("DEFAULT_SERVICE", ""),
		PromptPrefix:   "Demo:",
		FrontendTitle:  "Tiny LLM Chat",
	}

	flag.StringVar(&cfg.Addr, "addr", getenv("ADDR", cfg.Addr), "HTTP listen address")
	flag.StringVar(&cfg.CatalogPath, "catalog-path", getenv("CATALOG_PATH", cfg.CatalogPath), "path to services catalog")
	flag.StringVar(&cfg.DefaultService, "default-service", getenv("DEFAULT_SERVICE", cfg.DefaultService), "default backend service")
	flag.StringVar(&cfg.PromptPrefix, "prompt-prefix", getenv("PROMPT_PREFIX", cfg.PromptPrefix), "system prompt prefix")
	flag.StringVar(&cfg.FrontendTitle, "frontend-title", getenv("FRONTEND_TITLE", cfg.FrontendTitle), "frontend title")
	flag.Parse()

	return cfg
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
