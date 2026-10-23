# Diagramas del proyecto

- [contenedores-entrega2.mmd](contenedores-entrega2.mmd): flujo operativo actual, con gateway, Consumo, Electrodomésticos, MySQL y Jaeger. El mismo diagrama se incluye como bloque Mermaid en [arquitectura](../ARCHITECTURE.md).
- [contexto.mmd](contexto.mmd) y [contenedores.mmd](contenedores.mmd): contexto y arquitectura completa prevista para la final, con sus PNG de entrega 1.
- [contenedores-actuales.mmd](contenedores-actuales.mmd) y su PNG: fotografía histórica de los esqueletos y mock de entrega 1, conservada para reproducir [la arquitectura anterior](../ARCHITECTURE-ENTREGA-1.md).

No se presenta un PNG histórico como representación de los componentes actuales. El Mermaid E2 es editable y se renderiza directamente en visores compatibles con Markdown/GitHub. Para exportarlo opcionalmente con Mermaid CLI:

```sh
mmdc -i docs/diagrams/contenedores-entrega2.mmd -o docs/diagrams/contenedores-entrega2.png -b white -s 2 -c docs/diagrams/mermaid.config.json
```

La generación de documentación es independiente del arranque; Mermaid no es una dependencia de los servicios. La distribución actual se prueba además con la traza real de ocho spans incluida en docs/evidence/entrega2. Fuente: [Mermaid CLI](https://github.com/mermaid-js/mermaid-cli).
