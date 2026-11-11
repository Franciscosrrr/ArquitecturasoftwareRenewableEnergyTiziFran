package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"io"
	"log/slog"
	"renewable.local/platform"
	"renewable.local/solar/internal/adapters/persistence"
	"renewable.local/solar/internal/application"
	"renewable.local/solar/internal/domain"
	"time"
)

/**
 * Define la consulta comercial por modelo exacto que utiliza el worker solar.
 *
 * @method Supplier
 * @returns {Supplier} Representación tipada usada por este componente.
 */
type Supplier interface {
	/**
	 * Consulta la oferta del modelo exacto solicitado por el estudio.
	 *
	 * @method QuoteModel
	 * @param {context.Context} ctx - Presupuesto de la interacción comercial.
	 * @param {int64} watts - Potencia objetivo en W.
	 * @param {string} sku - Modelo local que se debe cotizar.
	 * @returns {domain.Quote, error} Oferta del modelo exacto o error de stock, dependencia o contrato.
	 */
	QuoteModel(context.Context, int64, string) (domain.Quote, error)
}

/**
 * Coordina consumo AMQP, deduplicación, cálculo y persistencia antes de confirmar el mensaje.
 *
 * @method Consumer
 * @returns {Consumer} Representación tipada usada por este componente.
 */
type Consumer struct {
	Store    *persistence.Store
	Supplier Supplier
}

/**
 * Representa ConsumoEvaluado.v1 y los datos congelados necesarios para crear un estudio.
 *
 * @method Event
 * @returns {Event} Representación tipada usada por este componente.
 */
type Event struct {
	ID           string              `json:"eventId"`
	Type         string              `json:"tipo"`
	StudyID      string              `json:"estudioId"`
	EvaluationID string              `json:"evaluacionId"`
	Owner        string              `json:"propietarioId"`
	Snapshot     json.RawMessage     `json:"snapshot"`
	Solar        domain.StudyOptions `json:"solar"`
	Traceparent  string              `json:"traceparent"`
	RequestID    string              `json:"requestId"`
}

/**
 * Rechaza eventos desconocidos, cuerpos sobredimensionados e identidades incompatibles.
 *
 * @method Parse
 * @param {[]byte} raw - Cuerpo original recibido para validar.
 * @returns {Event, error} Evento compatible o error permanente de mensaje que debe conservarse en DLQ.
 */
func Parse(raw []byte) (Event, error) {
	var e Event
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if len(raw) > 1048576 || d.Decode(&e) != nil || d.Decode(new(any)) != io.EOF || e.Type != "ConsumoEvaluado.v1" || len(e.ID) != 36 || len(e.StudyID) != 36 || len(e.Owner) != 36 || len(e.EvaluationID) != 36 {
		return e, &platform.PoisonError{Reason: "evento incompatible"}
	}
	return e, nil
}

/**
 * Deriva una identidad de contenido excluyendo correlación de transporte variable.
 *
 * @method eventHash
 * @param {Event} event - Evento de negocio que debe procesarse.
 * @returns {string, error} Huella canónica o error de evento no procesable.
 */
func eventHash(event Event) (string, error) {
	event.Traceparent = ""
	event.RequestID = ""
	var normalized any
	decoder := json.NewDecoder(bytes.NewReader(event.Snapshot))
	decoder.UseNumber()
	if decoder.Decode(&normalized) == nil {
		event.Snapshot, _ = json.Marshal(normalized)
	}
	canonical, err := json.Marshal(event)
	if err != nil {
		return "", &platform.PoisonError{Reason: "representación de evento inválida"}
	}
	return platform.Digest(string(canonical)), nil
}

/**
 * Coordina claim, reglas por puertos y persistencia del estudio antes del ACK.
 *
 * @method Handle
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {[]byte} raw - Cuerpo original recibido para validar.
 * @returns {error} nil después de persistir o detectar un duplicado; error para aplicar la política de mensajería.
 */
func (c Consumer) Handle(ctx context.Context, raw []byte) error {
	event, err := Parse(raw)
	if err != nil {
		return err
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{"traceparent": event.Traceparent})
	ctx, span := otel.Tracer("solar").Start(ctx, "solar.procesar-evaluacion")
	defer span.End()
	traceID := platform.TraceID(ctx)
	hash, err := eventHash(event)
	if err != nil {
		return err
	}
	claimed, err := c.Store.Claim(ctx, persistence.Study{ID: event.StudyID, EventID: event.ID, EvaluationID: event.EvaluationID, Owner: event.Owner, Hash: hash, Snapshot: event.Snapshot, TraceID: traceID})
	if err != nil {
		return err
	}
	if !claimed {
		platform.Operations.WithLabelValues("estudios", "duplicado_ignorado").Inc()
		return nil
	}
	completed := false
	defer func() {
		if !completed {
			cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = c.Store.Failed(cleanup, event.StudyID)
		}
	}()
	var snapshot struct {
		Consumption domain.Consumption `json:"consumo"`
	}
	if json.Unmarshal(event.Snapshot, &snapshot) != nil {
		return &platform.PoisonError{Reason: "snapshot inválido"}
	}
	panels, err := c.Store.Panels(ctx, false)
	if err != nil {
		return err
	}
	result, err := application.RunStudy(ctx, snapshot.Consumption.Daily, event.Solar, panels, c.Supplier)
	if errors.Is(err, domain.ErrValidation) {
		return &platform.PoisonError{Reason: "parámetros solares inválidos"}
	}
	if err != nil {
		return err
	}

	if err = c.Store.Complete(ctx, event.StudyID, result, traceID); err != nil {
		return err
	}
	completed = true
	platform.Operations.WithLabelValues("estudios", result.Status).Inc()
	slog.InfoContext(ctx, "solar.estudio.procesado", "study_id", event.StudyID, "trace_id", traceID, "estado", result.Status)
	return nil
}

/**
 * Mantiene un consumidor reconectable de los eventos de evaluación.
 *
 * @method Run
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func (c Consumer) Run(ctx context.Context) {
	platform.RetryLoop(ctx, func(ctx context.Context) error { return platform.Consume(ctx, "consumo.evaluado.v1", c.Handle) })
}
