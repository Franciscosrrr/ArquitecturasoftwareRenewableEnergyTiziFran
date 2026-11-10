package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Credencial ficticia exclusiva de pruebas; no es una configuración del servidor.
const testHealthToken = "credencial-ficticia-solo-para-pruebas-123456"

/**
 * Limita los contratos publicados por M2M aunque los otros archivos estén en la imagen.
 *
 * @method TestCapabilityOnlyContractSurface
 * @param {*testing.T} t - Contexto de prueba con archivos públicos y privados presentes.
 * @returns {void} Falla si M2M sirve otro contrato o si el sistema completo pierde sus documentos.
 */
func TestCapabilityOnlyContractSurface(t *testing.T) {
	docs := t.TempDir()
	contracts := map[string]string{"openapi.json": "v1", "guest-openapi.json": "guest-v1", "catalogo-openapi.json": "catalog-v1", "solar-openapi.json": "solar-v1", "paneles-openapi.json": "panel-search-v1", "privado-openapi.json": "private-v1"}
	fixture := []byte(`{"contrato":"fixture-presente-en-disco"}`)
	for _, folder := range contracts {
		if err := os.MkdirAll(filepath.Join(docs, folder), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(docs, folder, "openapi.json"), fixture, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, capabilityOnly := range []bool{false, true} {
		handler := NewHandler(testHealthToken, Config{ConsumoURL: "http://127.0.0.1:9", DocsDir: docs, WebDir: t.TempDir(), CapabilityOnly: capabilityOnly})
		for path := range contracts {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/"+path, nil))
			want := http.StatusOK
			if capabilityOnly && path != "openapi.json" {
				want = http.StatusNotFound
			}
			if response.Code != want {
				t.Fatalf("capabilityOnly=%t path=%s: status=%d want=%d", capabilityOnly, path, response.Code, want)
			}
			if want == http.StatusOK && response.Body.String() != string(fixture) {
				t.Fatalf("contenido inesperado: %s", path)
			}
		}
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por HealthRejectsUnauthenticatedRequests.
 *
 * @method TestHealthRejectsUnauthenticatedRequests
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestHealthRejectsUnauthenticatedRequests(t *testing.T) {
	handler := NewHandler(testHealthToken)
	for _, path := range []string{"/health/live", "/health/ready", "/health/ready/publico"} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
			for _, auth := range []string{"", "Bearer incorrecta", "Bearer mock-consumidor", "Basic " + testHealthToken} {
				t.Run(path+"/"+method+"/"+auth, func(t *testing.T) {
					request := httptest.NewRequest(method, path+"?token="+testHealthToken, nil)
					request.Header.Set("Authorization", auth)
					request.AddCookie(&http.Cookie{Name: "HEALTHCHECK_TOKEN", Value: testHealthToken})
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					if response.Code != http.StatusUnauthorized {
						t.Fatalf("status = %d, want 401", response.Code)
					}
					if response.Header().Get("WWW-Authenticate") == "" || response.Header().Get("Cache-Control") != "no-store" {
						t.Fatal("missing authentication challenge or cache protection")
					}
					if strings.Contains(response.Body.String(), "estado") || strings.Contains(response.Body.String(), testHealthToken) {
						t.Fatal("unauthenticated response exposes state or credential")
					}
				})
			}
		}
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por AuthenticatedHealthReportsActualReadiness.
 *
 * @method TestAuthenticatedHealthReportsActualReadiness
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestAuthenticatedHealthReportsActualReadiness(t *testing.T) {
	handler := NewHandler(testHealthToken)
	for _, test := range []struct {
		path  string
		code  int
		state string
	}{
		{"/health/live", http.StatusOK, "proceso-iniciado"},
		{"/health/ready", http.StatusServiceUnavailable, "negocio-no-implementado"},
	} {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+testHealthToken)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.code || body["estado"] != test.state || len(body) != 1 {
				t.Fatalf("unexpected health response: %d %v", response.Code, body)
			}
		})
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/pendiente", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("private business status = %d, want 401", response.Code)
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por InvalidHealthConfigurationFailsClosed.
 *
 * @method TestInvalidHealthConfigurationFailsClosed
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestInvalidHealthConfigurationFailsClosed(t *testing.T) {
	for _, token := range []string{"", "corta"} {
		request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		NewHandler(token).ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("invalid token allowed: %d", response.Code)
		}
	}
}
