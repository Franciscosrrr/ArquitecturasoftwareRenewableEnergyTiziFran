package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Credencial ficticia exclusiva de pruebas; no es una configuración del servidor.
const testHealthToken = "credencial-ficticia-solo-para-pruebas-123456"

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
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("business status = %d, want 501", response.Code)
	}
}

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
