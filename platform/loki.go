package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
)

/**
 * Mantiene la cola asíncrona de Loki que NewLogWriter combina con la salida stdout.
 *
 * @method LokiWriter
 * @returns {LokiWriter} Representación tipada usada por este componente.
 */
type LokiWriter struct{ queue chan []byte }

/**
 * Encola una copia del log sin bloquear el flujo de negocio.
 *
 * @method Write
 * @param {[]byte} raw - Registro de log que se copia al buffer de exportación.
 * @returns {int, error} len(raw) y nil; si la cola está llena descarta la copia y registra la métrica.
 */
func (l *LokiWriter) Write(raw []byte) (int, error) {
	copyRaw := append([]byte(nil), raw...)
	select {
	case l.queue <- copyRaw:
	default:
		Operations.WithLabelValues("logs", "descartado_cola_llena").Inc()
	}
	return len(raw), nil
}

/**
 * Replica logs a stdout y a un buffer acotado para exportación asíncrona.
 *
 * @method NewLogWriter
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {io.Writer} stdout - Salida local que debe recibir el log aunque falle la exportación.
 * @param {string} service - Nombre estable del servicio para trazas, logs y métricas.
 * @returns {io.Writer} Escritor que mantiene stdout y exporta a Loki con un buffer acotado.
 */
func NewLogWriter(ctx context.Context, stdout io.Writer, service string) io.Writer {
	endpoint := Env("LOKI_URL", "")
	if endpoint == "" {
		return stdout
	}
	l := &LokiWriter{queue: make(chan []byte, 1024)}
	go func() {
		client := &http.Client{Timeout: time.Second}
		for {
			select {
			case <-ctx.Done():
				return
			case raw := <-l.queue:
				values := [][]string{{strconv.FormatInt(time.Now().UnixNano(), 10), string(raw)}}
				for len(l.queue) > 0 && len(values) < 64 {
					values = append(values, []string{strconv.FormatInt(time.Now().UnixNano(), 10), string(<-l.queue)})
				}
				body, _ := json.Marshal(map[string]any{"streams": []map[string]any{{"stream": map[string]string{"service": service, "instance": Env("INSTANCE_ID", service)}, "values": values}}})
				req, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/loki/api/v1/push", bytes.NewReader(body))
				if err != nil {
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				res, err := client.Do(req)
				if err != nil {
					Operations.WithLabelValues("logs", "error_exportacion").Inc()
					continue
				}
				io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
				res.Body.Close()
				if res.StatusCode >= 300 {
					Operations.WithLabelValues("logs", "error_exportacion").Inc()
				}
			}
		}
	}()
	return io.MultiWriter(stdout, l)
}
