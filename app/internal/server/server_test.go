package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
	cfg := appconfig.Config{ModelMode: "mock", PromptPrefix: "Demo:", ServiceName: "tiny-llm"}
	h := New(cfg).Handler()

	configReq := httptest.NewRequest(http.MethodGet, "/config", nil)
	configRec := httptest.NewRecorder()
	h.ServeHTTP(configRec, configReq)
	if configRec.Code != http.StatusOK || !bytes.Contains(configRec.Body.Bytes(), []byte("Demo:")) {
		t.Fatalf("unexpected config response: %d %s", configRec.Code, configRec.Body.String())
	}

	genReq := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewBufferString(`{"prompt":"hello"}`))
	genRec := httptest.NewRecorder()
	h.ServeHTTP(genRec, genReq)
	if genRec.Code != http.StatusOK || !bytes.Contains(genRec.Body.Bytes(), []byte("hello")) {
		t.Fatalf("unexpected generate response: %d %s", genRec.Code, genRec.Body.String())
	}
}
