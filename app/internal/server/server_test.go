package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	appconfig "devops-demo/app/internal/config"
)

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	New(appconfig.Config{}).Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestConfigAndGenerateEndpoints(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected backend path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "hello back"}}},
		})
	}))
	defer backend.Close()

	tmp := t.TempDir()
	catalogPath := filepath.Join(tmp, "services.json")
	if err := os.WriteFile(catalogPath, []byte(`{"services":[{"name":"tiny-llm","namespace":"tiny-llm","backendUrl":"`+backend.URL+`","modelRepository":"SmolLM2/135M","modelFile":"model.gguf","modelRevision":"main","promptPrefix":"Demo:"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := appconfig.Config{CatalogPath: catalogPath, PromptPrefix: "Demo:", FrontendTitle: "Tiny LLM Chat"}
	h := New(cfg).Handler()

	configReq := httptest.NewRequest(http.MethodGet, "/config", nil)
	configRec := httptest.NewRecorder()
	h.ServeHTTP(configRec, configReq)
	if configRec.Code != http.StatusOK || !bytes.Contains(configRec.Body.Bytes(), []byte("Tiny LLM Chat")) {
		t.Fatalf("unexpected config response: %d %s", configRec.Code, configRec.Body.String())
	}

	servicesReq := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	servicesRec := httptest.NewRecorder()
	h.ServeHTTP(servicesRec, servicesReq)
	if servicesRec.Code != http.StatusOK || !bytes.Contains(servicesRec.Body.Bytes(), []byte("tiny-llm")) {
		t.Fatalf("unexpected services response: %d %s", servicesRec.Code, servicesRec.Body.String())
	}

	genReq := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewBufferString(`{"prompt":"hello"}`))
	genRec := httptest.NewRecorder()
	h.ServeHTTP(genRec, genReq)
	if genRec.Code != http.StatusOK || !bytes.Contains(genRec.Body.Bytes(), []byte("hello back")) {
		t.Fatalf("unexpected generate response: %d %s", genRec.Code, genRec.Body.String())
	}
}
