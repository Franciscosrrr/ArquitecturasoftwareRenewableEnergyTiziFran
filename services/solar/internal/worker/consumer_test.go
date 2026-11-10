package worker

import (
	"encoding/json"
	"renewable.local/solar/internal/domain"
	"testing"
)

/**
 * Comprueba que el orden de claves y la correlación no cambian la identidad del evento.
 *
 * @method TestCanonicalEventHash
 * @param {*testing.T} t - Contexto de verificación.
 * @returns {void} Falla si una retransmisión equivalente se identifica como otro contenido.
 */
func TestCanonicalEventHash(t *testing.T) {
	a := Event{ID: "id", Solar: domain.StudyOptions{Sun: "4", Yield: "0.8", Coverage: "1"}, Snapshot: json.RawMessage(`{"consumo":{"total":1,"diario":2},"lugar":{"id":"x"}}`)}
	b := a
	b.Traceparent = "otra-traza"
	b.RequestID = "otro-request"
	b.Snapshot = json.RawMessage(`{ "lugar":{"id":"x"},"consumo":{"diario":2,"total":1} }`)
	ah, ae := eventHash(a)
	bh, be := eventHash(b)
	if ae != nil || be != nil || ah != bh {
		t.Fatal("hash depende del orden o correlación")
	}
	b.Owner = "otro"
	bh, be = eventHash(b)
	if be != nil || ah == bh {
		t.Fatal("hash ignora propietario")
	}
}

/**
 * Verifica que eventos ajenos a la versión soportada no lleguen a persistencia.
 *
 * @method TestIncompatibleEventIsPoison
 * @param {*testing.T} t - Contexto de verificación.
 * @returns {void} Falla si el parser acepta una versión desconocida.
 */
func TestIncompatibleEventIsPoison(t *testing.T) {
	if _, err := Parse([]byte(`{"tipo":"Version.v99"}`)); err == nil {
		t.Fatal("aceptó versión desconocida")
	}
}
