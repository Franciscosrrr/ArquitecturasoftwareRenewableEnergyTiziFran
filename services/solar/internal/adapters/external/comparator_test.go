package external

import (
	"context"
	"errors"
	"fmt"
	"github.com/goccy/go-yaml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"renewable.local/solar/internal/domain"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

/**
 * Entrega fichas técnicas propias durante la prueba del contrato comercial.
 */
type comparatorCatalog struct{}

/**
 * Recupera la referencia técnica local sin derivar potencia de un SKU.
 *
 * @method Panels
 * @param {context.Context} ctx - Contexto de lectura.
 * @param {bool} all - Incluye fichas inactivas.
 * @returns {[]domain.Panel, error} Ficha utilizada por el consumidor.
 */
func (comparatorCatalog) Panels(ctx context.Context, all bool) ([]domain.Panel, error) {
	return []domain.Panel{{ID: "PANEL-SIM-550", Watts: 550, AreaMilli: 2800, Active: true}}, nil
}

const commercialOffer = `[{"proveedor_id":3,"proveedor_nombre":"Cordoba Energia SA","precio_unitario":189573.99,"descuento_pct":15,"descuento_aplicado":false,"flete":43000,"stock_disponible":394,"costo_total":422147.98}]`

/**
 * Verifica consulta, autenticación, cantidad local e importes exactos con flete y descuento.
 *
 * @method TestComparatorCommercialContract
 * @param {*testing.T} t - Prueba del consumidor contra respuestas acordadas.
 * @returns {void} Falla ante una adaptación o validación incompatible.
 */
func TestComparatorCommercialContract(t *testing.T) {
	for _, tc := range []struct {
		quantity int64
		total    string
		applied  bool
	}{{2, "422147.98", false}, {10, "1654378.92", true}} {
		t.Run(fmt.Sprint(tc.quantity), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/api/v1/proveedores/recomendados" || r.URL.Query().Get("producto") != "PAN-550W-MONO" || r.URL.Query().Get("cantidad") != fmt.Sprint(tc.quantity) || r.URL.Query().Get("limite") != "1" || r.Header.Get("X-API-Key") != "test-private-key" || r.Header.Get("Authorization") != "" {
					t.Error("solicitud fuera del contrato")
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Request-Id", "provider-request-123")
				body := strings.Replace(commercialOffer, "422147.98", tc.total, 1)
				if tc.applied {
					body = strings.Replace(body, `"descuento_aplicado":false`, `"descuento_aplicado":true`, 1)
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			client, _ := NewClient(server.URL, "test-private-key", time.Second)
			c := Comparator{Client: client, Catalog: comparatorCatalog{}}
			q, err := c.QuoteModel(context.Background(), tc.quantity*550, "PANEL-SIM-550")
			if err != nil || q.ID != "" || q.Source != "externo" || q.Quantity != tc.quantity || q.UnitCents != 18957399 || q.FreightCents != 4300000 || q.RequestID != "provider-request-123" {
				t.Fatalf("%+v %v", q, err)
			}
			if tc.quantity == 2 && q.TotalCents != 42214798 || tc.quantity == 10 && q.TotalCents != 165437892 {
				t.Fatal("importe alterado", q.TotalCents)
			}
		})
	}
}

/**
 * Detecta cambios de forma, tipos, campos obligatorios, decimales y semántica monetaria.
 *
 * @method TestComparatorRejectsBreakingResponses
 * @param {*testing.T} t - Prueba de incompatibilidades del proveedor.
 * @returns {void} Falla si una respuesta inválida se acepta como oferta real.
 */
func TestComparatorRejectsBreakingResponses(t *testing.T) {
	cases := []string{`null`, `{}`, `[{"proveedor_id":3}]`, strings.Replace(commercialOffer, `"descuento_aplicado":false,`, "", 1), strings.Replace(commercialOffer, `189573.99`, `"189573.99"`, 1), strings.Replace(commercialOffer, `189573.99`, `189573.991`, 1), strings.Replace(commercialOffer, `"stock_disponible":394`, `"stock_disponible":"394"`, 1), strings.Replace(commercialOffer, `422147.98`, `379147.98`, 1), strings.Replace(commercialOffer, `"descuento_pct":15`, `"descuento_pct":150`, 1)}
	for i, body := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Request-Id", "test-request")
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			client, _ := NewClient(server.URL, "test-private-key", time.Second)
			c := Comparator{Client: client, Catalog: comparatorCatalog{}}
			if _, err := c.QuoteModel(context.Background(), 1100, "PANEL-SIM-550"); !errors.Is(err, domain.ErrContract) {
				t.Fatalf("%d: %v", i, err)
			}
		})
	}
}

/**
 * Distingue ausencia de stock, SKU inexistente, autenticación y espera obligatoria por cuota.
 *
 * @method TestComparatorErrorsAndRetryAfter
 * @param {*testing.T} t - Casos de errores públicos del proveedor.
 * @returns {void} Falla si se confunde stock con una caída o se adelanta un reintento.
 */
func TestComparatorErrorsAndRetryAfter(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   error
	}{{200, "[]", domain.ErrStock}, {404, `{"code":"PRODUCTO_NO_ENCONTRADO"}`, domain.ErrStock}, {404, `{"code":"RUTA_NO_ENCONTRADA"}`, domain.ErrContract}, {401, `{"code":"NO_AUTORIZADO"}`, domain.ErrContract}, {400, `{"code":"PARAMETRO_INVALIDO"}`, domain.ErrContract}, {429, `{"code":"LIMITE_EXCEDIDO"}`, domain.ErrUnavailable}, {503, `{"code":"NO_DISPONIBLE"}`, domain.ErrUnavailable}} {
		t.Run(fmt.Sprint(tc.status, tc.body), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/problem+json")
				if tc.status == 200 {
					w.Header().Set("Content-Type", "application/json")
				}
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			client, _ := NewClient(server.URL, "test-private-key", time.Second)
			c := Comparator{Client: client, Catalog: comparatorCatalog{}}
			_, err := c.QuoteModel(context.Background(), 1100, "PANEL-SIM-550")
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			if tc.status == 429 {
				_, _ = c.QuoteModel(context.Background(), 1100, "PANEL-SIM-550")
				if calls.Load() != 1 || time.Until(c.until) < 59*time.Second {
					t.Fatal("Retry-After ignorado")
				}
			}
		})
	}
}

/**
 * Comprueba tiempo de espera, circuito tras tres fallos y prueba de recuperación.
 *
 * @method TestComparatorTimeoutCircuitAndRecovery
 * @param {*testing.T} t - Ensayo técnico con servidor controlado.
 * @returns {void} Falla ante llamadas ilimitadas o falta de recuperación.
 */
func TestComparatorTimeoutCircuitAndRecovery(t *testing.T) {
	var calls atomic.Int32
	var healthy atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !healthy.Load() {
			<-r.Context().Done()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "recovered")
		fmt.Fprint(w, commercialOffer)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "test-private-key", 30*time.Millisecond)
	c := Comparator{Client: client, Catalog: comparatorCatalog{}}
	for i := 0; i < 4; i++ {
		if _, err := c.QuoteModel(context.Background(), 1100, "PANEL-SIM-550"); err != domain.ErrUnavailable {
			t.Fatal(err)
		}
	}
	if calls.Load() != 3 {
		t.Fatal("circuito no abierto", calls.Load())
	}
	healthy.Store(true)
	c.until = time.Now().Add(-time.Second)
	if _, err := c.QuoteModel(context.Background(), 1100, "PANEL-SIM-550"); err != nil {
		t.Fatal("no recuperó", err)
	}
}

/**
 * Evita reenviar la clave privada a una redirección del servidor externo.
 *
 * @method TestComparatorDoesNotRedirectSecrets
 * @param {*testing.T} t - Prueba de aislamiento de credenciales.
 * @returns {void} Falla si se sigue una redirección comercial.
 */
func TestComparatorDoesNotRedirectSecrets(t *testing.T) {
	var leaked atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Store(true) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer server.Close()
	client, _ := NewClient(server.URL, "test-private-key", time.Second)
	c := Comparator{Client: client, Catalog: comparatorCatalog{}}
	if _, err := c.QuoteModel(context.Background(), 1100, "PANEL-SIM-550"); err != domain.ErrContract || leaked.Load() {
		t.Fatal("redirección insegura", err)
	}
}

/**
 * Verifica el OpenAPI congelado o una copia nueva contra las necesidades reales del consumidor.
 *
 * @method TestComparatorOpenAPICompatibility
 * @param {*testing.T} t - Contrato del consumidor con control de cambios incompatibles.
 * @returns {void} Falla si cambian método, ruta, autenticación, forma, campos o tipos.
 */
func TestComparatorOpenAPICompatibility(t *testing.T) {
	path := os.Getenv("PANEL_COMPARATOR_OPENAPI")
	if path == "" {
		path = filepath.FromSlash("../../../../../docs/contracts/comparador-proveedores-v1.1.0/openapi.yaml")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		OpenAPI string `yaml:"openapi"`
		Info    struct {
			Version     string `yaml:"version"`
			Description string `yaml:"description"`
		} `yaml:"info"`
		Paths map[string]struct {
			Get struct {
				Parameters []struct {
					Ref string `yaml:"$ref"`
				} `yaml:"parameters"`
				Responses map[string]struct {
					Content map[string]struct {
						Schema struct {
							Type  string `yaml:"type"`
							Items struct {
								Ref string `yaml:"$ref"`
							} `yaml:"items"`
						} `yaml:"schema"`
					} `yaml:"content"`
				} `yaml:"responses"`
			} `yaml:"get"`
		} `yaml:"paths"`
		Components struct {
			Parameters map[string]struct {
				Name     string `yaml:"name"`
				In       string `yaml:"in"`
				Required bool   `yaml:"required"`
				Schema   struct {
					Type string `yaml:"type"`
				} `yaml:"schema"`
			} `yaml:"parameters"`
			Schemas map[string]struct {
				Required   []string `yaml:"required"`
				Properties map[string]struct {
					Type string `yaml:"type"`
				} `yaml:"properties"`
			} `yaml:"schemas"`
			Security map[string]struct {
				Type string `yaml:"type"`
				In   string `yaml:"in"`
				Name string `yaml:"name"`
			} `yaml:"securitySchemes"`
		} `yaml:"components"`
	}
	if err = yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	operation := doc.Paths["/api/v1/proveedores/recomendados"].Get
	schema := operation.Responses["200"].Content["application/json"].Schema
	auth := doc.Components.Security["ApiKeyAuth"]
	if doc.OpenAPI != "3.0.3" || !strings.HasPrefix(doc.Info.Version, "1.") || schema.Type != "array" || schema.Items.Ref != "#/components/schemas/Recomendacion" || auth.Type != "apiKey" || auth.In != "header" || auth.Name != "X-API-Key" || !strings.Contains(doc.Info.Description, "ARS") {
		t.Fatal("operación o autenticación incompatible")
	}
	refs := map[string]bool{}
	for _, p := range operation.Parameters {
		refs[p.Ref] = true
	}
	for key, name := range map[string]string{"Producto": "producto", "Cantidad": "cantidad", "Limite": "limite"} {
		p := doc.Components.Parameters[key]
		kind := "integer"
		if key == "Producto" {
			kind = "string"
		}
		if !refs["#/components/parameters/"+key] || p.Name != name || p.In != "query" || p.Schema.Type != kind || key == "Producto" && !p.Required {
			t.Error("parámetro incompatible", key)
		}
	}
	fields := map[string]string{"proveedor_id": "integer", "proveedor_nombre": "string", "precio_unitario": "number", "descuento_pct": "number", "descuento_aplicado": "boolean", "flete": "number", "stock_disponible": "integer", "costo_total": "number"}
	model := doc.Components.Schemas["Recomendacion"]
	required := map[string]bool{}
	for _, key := range model.Required {
		required[key] = true
	}
	for key, kind := range fields {
		if !required[key] || model.Properties[key].Type != kind {
			t.Error("campo incompatible", key)
		}
	}
	for _, status := range []string{"200", "400", "401", "404", "429", "502", "503", "504"} {
		if _, ok := operation.Responses[status]; !ok {
			t.Error("estado eliminado", status)
		}
	}
}
