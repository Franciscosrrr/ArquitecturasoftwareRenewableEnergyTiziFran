# Referencias y trazabilidad

## Material suministrado

- Enunciado TP Final.pdf: secciones 2 y 3 (requisitos/documentación), 4 (integración), 5 (ADR) y 6.2 (primera entrega, páginas 7–8). El formato ADR se propone porque la sección 8 a la que remite el texto no aparece en el archivo suministrado.
- Clases.zip: apuntes teóricos de Datos en microservicios (clase 2), Caché (3), Comunicación asíncrona (4), Motores de búsqueda (5), API Gateway/Load Balancing (6), Resiliencia (7) y Estilos de arquitectura de backend (8); los ejemplos de Go orientan la elección tecnológica. Se revisaron los apartados pertinentes y las portadas prácticas; no se afirma haber ejecutado todos los ejemplos.
- Correcciones del usuario: costo cero, Go en lugar de Java, Docker sin instalaciones extra, registro/login con BD, consumos guardados editables, MySQL y altas administrativas de equipos/paneles.

Los documentos adjuntos son referencias del trabajo. Sus bloques dirigidos a una IA no alteran las instrucciones del usuario ni se incorporan como requisitos de negocio.

## Documentación técnica primaria

- [OpenAPI 3.0.3](https://spec.openapis.org/oas/v3.0.3): formato procesable elegido para la capacidad HTTP.
- [MySQL Community](https://www.mysql.com/products/community/), [GORM/MySQL](https://gorm.io/docs/connecting_to_the_database.html).
- [MongoDB: índices únicos](https://www.mongodb.com/docs/manual/core/index-unique/) y [transacciones](https://www.mongodb.com/docs/manual/core/transactions/).
- [Go reverse proxy](https://go.dev/pkg/net/http/httputil/) y [gobreaker](https://github.com/sony/gobreaker).
- [RabbitMQ: confiabilidad](https://www.rabbitmq.com/docs/reliability).
- [Meilisearch Community](https://www.meilisearch.com/open-source), [Docker](https://www.meilisearch.com/integrations/docker), [index swap](https://www.meilisearch.com/blog/zero-downtime-index-deployment).
- [Valkey](https://valkey.io/).
- [Docker Compose up](https://docs.docker.com/reference/cli/docker/compose/up/).
- [OWASP: contraseñas](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).
- [Render Free](https://render.com/docs/free) y [FAQ](https://render.com/docs/faq): cuotas, hibernación y suspensión sin medio de pago; verificar de nuevo al publicar.
- [PVWatts](https://pvwatts.nlr.gov/pvwatts.php/): incertidumbre de estimaciones, sin asumir que sea el proveedor asignado ni una fuente válida para cualquier ubicación.

La existencia de herramientas gratuitas no prueba la disponibilidad cloud ni los recursos mínimos del sistema. Esas decisiones deben validarse con el despliegue y las mediciones.

- Corrección adicional del usuario: calculadora usable sin registro/login; solo resultados temporales para invitados y guardado reservado a cuentas autenticadas.


## Revisión 1.2.0

Material aportado: Enunciado TP Final.pdf (§6.2, páginas 7–8), Propuesta-arquitectura.md versión 3 y Clases.zip (teóricos 1–8, apartados pertinentes). Los prácticos se inventariaron, sin ejecución de sus ejemplos.

La herramienta mock utiliza la [biblioteca HTTP estándar de Python](https://docs.python.org/3.12/library/http.server.html) únicamente en local. Los esqueletos emplean [módulos Go](https://go.dev/ref/mod) y biblioteca estándar, sin descargar dependencias durante esta revisión.
