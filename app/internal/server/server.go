package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	appconfig "devops-demo/app/internal/config"
)

type Server struct {
	cfg    appconfig.Config
	random *rand.Rand
}

func New(cfg appconfig.Config) *Server {
	return &Server{
		cfg:    cfg,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/generate", s.generate)
	mux.HandleFunc("/slow", s.slow)
	mux.HandleFunc("/error", s.fail)
	mux.HandleFunc("/config", s.config)
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) generate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Prompt string `json:"prompt"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Prompt == "" {
		req.Prompt = "hello"
	}

	response := fmt.Sprintf("%s %s -> mock completion", s.cfg.PromptPrefix, req.Prompt)
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":     s.cfg.ModelMode,
		"prompt":   req.Prompt,
		"response": response,
	})
}

func (s *Server) slow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	delay := time.Duration(1+s.random.Intn(3)) * time.Second
	select {
	case <-time.After(delay):
		writeJSON(w, http.StatusOK, map[string]any{"delaySeconds": delay.Seconds()})
	case <-r.Context().Done():
		w.WriteHeader(http.StatusRequestTimeout)
	}
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "intentional failure"})
}

func (s *Server) config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"serviceName":  s.cfg.ServiceName,
		"modelMode":    s.cfg.ModelMode,
		"promptPrefix": s.cfg.PromptPrefix,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Run(ctx context.Context, cfg appconfig.Config) error {
	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: New(cfg).Handler(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
