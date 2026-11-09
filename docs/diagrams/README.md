# Diagramas vigentes

Actualización 2026-11-09. Las fuentes Mermaid y PNG representan el mismo alcance. Las vistas Entrega1 y Entrega2 permanecen históricas; no describen el entorno final.

| Vista | Fuente | Imagen |
|---|---|---|
| Contexto y actores | [contexto.mmd](contexto.mmd) | [contexto.png](contexto.png) |
| Contenedores del sistema completo | [contenedores-actuales.mmd](contenedores-actuales.mmd) | [contenedores-actuales.png](contenedores-actuales.png) |
| Alias de arquitectura general | [contenedores.mmd](contenedores.mmd) | [contenedores.png](contenedores.png) |
| Evaluación durable y estados de publicación | [evaluacion-durable.mmd](evaluacion-durable.mmd) | [evaluacion-durable.png](evaluacion-durable.png) |
| Capacidad M2M cloud desplegada | [cloud-gratuito.mmd](cloud-gratuito.mmd) | [cloud-gratuito.png](cloud-gratuito.png) |

La vista cloud representa la unidad M2M publicada en Render; no representa el alojamiento de toda la arquitectura. Las bases son privadas por servicio. El vendedor es simulado y las llamadas salen de Solar directamente, sin atravesar gateway ni frontend. Los dos workers compiten por la misma cola; no procesan obligatoriamente cada evento ambos. El ACK ocurre después de persistir.

Regenerar con Mermaid CLI12.2.0: `mmdc -i docs/diagrams/contenedores-actuales.mmd -o docs/diagrams/contenedores-actuales.png -b white -s 2 -c docs/diagrams/mermaid.config.json`. Repetir para las demás fuentes. Las imágenes históricas no se regeneran con el modelo actual.

Publicación posterior de M2M por túnel: [fuente](publicacion-cloudflare.mmd), [imagen](publicacion-cloudflare.png) y [condiciones operativas](../PUBLICACION-CLINICA.md). La vista cloud-gratuito corresponde al M2M Render verificado, con evidencia en DESPLIEGUE-GRATUITO.md.
