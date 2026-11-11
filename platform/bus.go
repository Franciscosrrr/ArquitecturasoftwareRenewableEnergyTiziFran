package platform

import (
	"context"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"net"
	"strconv"
	"sync"
	"time"
)

/**
 * Agrupa conexión y canal AMQP, devoluciones del broker y exclusión del publicador.
 *
 * @method Bus
 * @returns {Bus} Representación tipada usada por este componente.
 */
type Bus struct {
	Conn     *amqp.Connection
	Ch       *amqp.Channel
	returned chan amqp.Return
	mu       sync.Mutex
}

/**
 * Identifica un mensaje incompatible que debe ir a DLQ sin reintentos transitorios.
 *
 * @method PoisonError
 * @returns {PoisonError} Representación tipada usada por este componente.
 */
type PoisonError struct{ Reason string }

/**
 * Devuelve la representación pública del error sin incluir credenciales.
 *
 * @method Error
 * @returns {string} Descripción pública y saneada del error.
 */
func (e *PoisonError) Error() string { return e.Reason }

/**
 * Asocia un tipo de evento con su cola de trabajo.
 *
 * @method QueueFor
 * @param {string} key - Routing key que identifica el evento de dominio y su cola.
 * @returns {string} Nombre de la cola asociada a la routing key de dominio.
 */
func QueueFor(key string) string {
	if key == "consumo.evaluado.v1" {
		return "solar.estudios"
	}
	return "catalogo.indice"
}

/**
 * Abre AMQP y declara exchanges, colas durables y confirmaciones del publicador.
 *
 * @method OpenBus
 * @returns {*Bus, error} Conexión y canal con topología declarada; error si AMQP no puede inicializarse.
 */
func OpenBus() (*Bus, error) {
	conn, err := amqp.DialConfig(Env("AMQP_URL", ""), amqp.Config{Heartbeat: 10 * time.Second, Dial: func(network, address string) (net.Conn, error) {
		return net.DialTimeout(network, address, 2*time.Second)
	}})
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	b := &Bus{Conn: conn, Ch: ch, returned: make(chan amqp.Return, 8)}
	ch.NotifyReturn(b.returned)
	for _, exchange := range []string{"renewable.events", "renewable.retry", "renewable.dlx"} {
		if err = ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
			b.Close()
			return nil, err
		}
	}
	for _, key := range []string{"consumo.evaluado.v1", "catalogo.actualizado.v1"} {
		q := QueueFor(key)
		for _, suffix := range []string{"", ".retry", ".dlq"} {
			args := amqp.Table{}
			exchange := "renewable.events"
			if suffix == "" {
				args["x-dead-letter-exchange"] = "renewable.dlx"
				args["x-dead-letter-routing-key"] = key
			}
			if suffix == ".retry" {
				args["x-dead-letter-exchange"] = "renewable.events"
				args["x-dead-letter-routing-key"] = key
				exchange = "renewable.retry"
			}
			if suffix == ".dlq" {
				exchange = "renewable.dlx"
			}
			if _, err = ch.QueueDeclare(q+suffix, true, false, false, false, args); err != nil {
				b.Close()
				return nil, err
			}
			if err = ch.QueueBind(q+suffix, key, exchange, false, nil); err != nil {
				b.Close()
				return nil, err
			}
		}
	}
	if err = ch.Confirm(false); err != nil {
		b.Close()
		return nil, err
	}
	return b, nil
}

/**
 * Libera canales y conexiones de infraestructura.
 *
 * @method Close
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func (b *Bus) Close() { _ = b.Ch.Close(); _ = b.Conn.Close() }

/**
 * Publica un mensaje persistente y espera confirmación y ausencia de devolución.
 *
 * @method Publish
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} exchange - Exchange AMQP al que se publica el evento.
 * @param {string} key - Routing key que identifica el evento de dominio y su cola.
 * @param {[]byte} raw - Cuerpo original recibido para validar.
 * @param {amqp.Table} headers - Metadatos AMQP, incluidos contador de intento y correlación cuando corresponda.
 * @param {string} expiration - Tiempo de expiración AMQP en milisegundos, o vacío para no fijarlo.
 * @returns {error} nil tras publisher confirm sin devolución; error si no se confirma la publicación.
 */
func (b *Bus) Publish(ctx context.Context, exchange, key string, raw []byte, headers amqp.Table, expiration string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for len(b.returned) > 0 {
		<-b.returned
	}
	confirmation, err := b.Ch.PublishWithDeferredConfirmWithContext(ctx, exchange, key, true, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: raw, Headers: headers, Expiration: expiration, Timestamp: time.Now()})
	if err != nil {
		return err
	}
	if confirmation == nil {
		return errors.New("confirmación ausente")
	}
	ok, err := confirmation.WaitContext(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("publicación rechazada")
	}
	select {
	case <-b.returned:
		return errors.New("evento sin destino")
	default:
		return nil
	}
}

/**
 * Reconecta una tarea cancelable con espera exponencial acotada.
 *
 * @method RetryLoop
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {func(context.Context) error} run - Tarea cancelable que debe reconectarse ante errores transitorios.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func RetryLoop(ctx context.Context, run func(context.Context) error) {
	delay := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		err := run(ctx)
		if err == nil || time.Since(started) >= 15*time.Second {
			delay = time.Second
		}
		if err != nil {
			Operations.WithLabelValues("broker", "reconexion").Inc()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		delay = min(delay*2, 15*time.Second)
	}
}

/**
 * Procesa entregas con ACK manual y transfiere fallas a retry o DLQ antes de confirmar.
 *
 * @method Consume
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} key - Routing key que identifica el evento de dominio y su cola.
 * @param {func(context.Context, []byte) error} handle - Procesador del mensaje; su resultado decide ACK, reintento o DLQ.
 * @returns {error} Error de conexión o consumo; nil al finalizar por cancelación cuando corresponde.
 */
func Consume(ctx context.Context, key string, handle func(context.Context, []byte) error) error {
	b, err := OpenBus()
	if err != nil {
		return err
	}
	defer b.Close()
	consumer, err := b.Conn.Channel()
	if err != nil {
		return err
	}
	defer consumer.Close()
	if err = consumer.Qos(1, 0, false); err != nil {
		return err
	}
	deliveries, err := consumer.Consume(QueueFor(key), "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("cola cerrada")
			}
			work, cancel := context.WithTimeout(ctx, 12*time.Second)
			err = handle(work, d.Body)
			cancel()
			if err == nil {
				_ = d.Ack(false)
				Operations.WithLabelValues(key, "ack").Inc()
				continue
			}
			attempts := int32(0)
			if v, ok := d.Headers["attempt"].(int32); ok {
				attempts = v
			}
			var poison *PoisonError
			exchange := "renewable.retry"
			expiration := strconv.Itoa([]int{1000, 5000, 30000}[min(int(attempts), 2)])
			reason := "dependencia_no_disponible"
			if errors.As(err, &poison) || attempts >= 3 {
				exchange = "renewable.dlx"
				expiration = ""
				reason = "mensaje_no_procesable"
			}
			headers := amqp.Table{"attempt": attempts + 1, "reason": reason}
			publish, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = b.Publish(publish, exchange, key, d.Body, headers, expiration)
			cancel()
			if err != nil {
				_ = d.Nack(false, true)
				return err
			}
			_ = d.Ack(false)
			Operations.WithLabelValues(key, exchange).Inc()
		}
	}
}
