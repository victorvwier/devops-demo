package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	appconfig "devops-demo/app/internal/config"
)

type ServiceCatalog struct {
	Services []ServiceEntry `json:"services"`
}

type ServiceEntry struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	BackendURL      string `json:"backendUrl"`
	PromptPrefix    string `json:"promptPrefix,omitempty"`
	ModelRepository string `json:"modelRepository,omitempty"`
	ModelFile       string `json:"modelFile,omitempty"`
	ModelRevision   string `json:"modelRevision,omitempty"`
}

type chatRequest struct {
	Service string `json:"service,omitempty"`
	Prompt  string `json:"prompt"`
}

type chatResponse struct {
	Service string `json:"service"`
	Prompt  string `json:"prompt"`
	Reply   string `json:"reply"`
}

type Server struct {
	cfg        appconfig.Config
	httpClient *http.Client
	logger     *slog.Logger
}

func New(cfg appconfig.Config) *Server {
	return &Server{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     slog.Default().With("component", "frontend"),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/api/services", s.services)
	mux.HandleFunc("/generate", s.generate)
	mux.HandleFunc("/api/chat", s.generate)
	mux.HandleFunc("/slow", s.slow)
	mux.HandleFunc("/error", s.fail)
	mux.HandleFunc("/config", s.config)
	return s.requestLogger(mux)
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

	var req chatRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Prompt == "" {
		req.Prompt = "hello"
	}

	entry, err := s.pickService(req.Service)
	if err != nil {
		s.logger.Warn("chat request rejected", "service", req.Service, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.logger.Info("chat request", "service", entry.Name, "modelRepository", entry.ModelRepository, "modelFile", entry.ModelFile)
	reply, err := s.chatWithBackend(r.Context(), entry, req.Prompt)
	if err != nil {
		s.logger.Error("backend request failed", "service", entry.Name, "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	s.logger.Info("chat complete", "service", entry.Name, "promptBytes", len(req.Prompt), "replyBytes", len(reply))

	writeJSON(w, http.StatusOK, chatResponse{
		Service: entry.Name,
		Prompt:  req.Prompt,
		Reply:   reply,
	})
}

func (s *Server) slow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	delay := time.Second + time.Duration(time.Now().UnixNano()%3)*time.Second
	s.logger.Info("slow request", "delaySeconds", delay.Seconds())
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
	s.logger.Info("intentional error requested")
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "intentional failure"})
}

func (s *Server) config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	catalog, _ := s.loadCatalog()
	writeJSON(w, http.StatusOK, map[string]string{
		"frontendTitle": s.cfg.FrontendTitle,
		"catalogPath":   s.cfg.CatalogPath,
		"services":      fmt.Sprintf("%d", len(catalog.Services)),
	})
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, indexHTML)
}

func (s *Server) services(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	catalog, err := s.loadCatalog()
	if err != nil {
		s.logger.Error("loading service catalog", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.logger.Info("served service catalog", "services", len(catalog.Services))
	writeJSON(w, http.StatusOK, catalog)
}

func (s *Server) loadCatalog() (ServiceCatalog, error) {
	data, err := os.ReadFile(s.cfg.CatalogPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ServiceCatalog{}, nil
		}
		return ServiceCatalog{}, err
	}
	var catalog ServiceCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return ServiceCatalog{}, err
	}
	sort.Slice(catalog.Services, func(i, j int) bool { return catalog.Services[i].Name < catalog.Services[j].Name })
	return catalog, nil
}

func (s *Server) pickService(name string) (ServiceEntry, error) {
	catalog, err := s.loadCatalog()
	if err != nil {
		return ServiceEntry{}, err
	}
	if name == "" {
		name = s.cfg.DefaultService
	}
	if name != "" {
		for _, entry := range catalog.Services {
			if entry.Name == name {
				return entry, nil
			}
		}
	}
	if len(catalog.Services) > 0 {
		return catalog.Services[0], nil
	}
	if backend := strings.TrimSpace(os.Getenv("BACKEND_URL")); backend != "" {
		return ServiceEntry{Name: "default", BackendURL: backend, PromptPrefix: s.cfg.PromptPrefix}, nil
	}
	return ServiceEntry{}, fmt.Errorf("no model backends are configured")
}

func (s *Server) chatWithBackend(ctx context.Context, entry ServiceEntry, prompt string) (string, error) {
	backendURL := strings.TrimRight(entry.BackendURL, "/")
	if backendURL == "" {
		return "", fmt.Errorf("service %q has no backend url", entry.Name)
	}

	body := map[string]any{
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   false,
	}
	if prefix := strings.TrimSpace(entry.PromptPrefix); prefix != "" {
		body["messages"] = []map[string]string{{"role": "system", "content": prefix}, {"role": "user", "content": prompt}}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	endpoint := backendURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("backend returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &parsed); err == nil && len(parsed.Choices) > 0 && parsed.Choices[0].Message.Content != "" {
		return parsed.Choices[0].Message.Content, nil
	}

	var fallback map[string]any
	if err := json.Unmarshal(data, &fallback); err == nil {
		if content, ok := fallback["response"].(string); ok && content != "" {
			return content, nil
		}
	}
	return strings.TrimSpace(string(data)), nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
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

func backendURLFor(namespace, name string) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local", name, namespace)
}

func normalizeBackendURL(raw string) string {
	return strings.TrimRight(raw, "/")
}

func serviceCatalogFromURLs(entries ...ServiceEntry) ServiceCatalog {
	return ServiceCatalog{Services: entries}
}

func resolveURL(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return parsed.String()
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Tiny LLM Chat</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, sans-serif; margin: 0; background: #0f172a; color: #e2e8f0; }
    main { max-width: 900px; margin: 0 auto; padding: 24px; }
    .card { background: #111827; border: 1px solid #334155; border-radius: 16px; padding: 16px; margin-bottom: 16px; }
    textarea, select, button { width: 100%; box-sizing: border-box; margin-top: 8px; padding: 12px; border-radius: 12px; border: 1px solid #475569; background: #0b1220; color: #e2e8f0; }
    button { background: #38bdf8; color: #082f49; font-weight: 700; cursor: pointer; }
    pre { white-space: pre-wrap; word-break: break-word; }
    .row { display: grid; gap: 12px; grid-template-columns: 1fr 1fr; }
    @media (max-width: 700px) { .row { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
<main>
  <div class="card">
    <h1>Tiny LLM Chat</h1>
    <p>Pick a model and chat with it from the shared frontend.</p>
  </div>
  <div class="card">
    <label>Model</label>
    <select id="service"></select>
    <label>Message</label>
    <textarea id="prompt" rows="5" placeholder="Ask something small and weird."></textarea>
    <button id="send">Send</button>
  </div>
  <div class="card">
    <h2>Reply</h2>
    <pre id="reply">Waiting...</pre>
  </div>
</main>
<script>
const serviceEl = document.getElementById('service');
const promptEl = document.getElementById('prompt');
const replyEl = document.getElementById('reply');
const sendEl = document.getElementById('send');

async function loadServices() {
  const res = await fetch('/api/services');
  const data = await res.json();
  serviceEl.innerHTML = '';
    (data.services || []).forEach((service) => {
    const opt = document.createElement('option');
    opt.value = service.name;
    opt.textContent = service.name + ' (' + (service.modelRepository || 'model') + ')';
    serviceEl.appendChild(opt);
  });
}

sendEl.addEventListener('click', async () => {
  replyEl.textContent = 'Thinking...';
  const res = await fetch('/api/chat', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({service: serviceEl.value, prompt: promptEl.value}),
  });
  const data = await res.json();
  replyEl.textContent = data.reply || data.error || JSON.stringify(data, null, 2);
});

loadServices().catch((err) => { replyEl.textContent = err.message; });
</script>
</body>
</html>`
