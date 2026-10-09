# Diagramas del proyecto

- contexto.mmd: actores y límite del sistema previsto.
- contenedores.mmd: arquitectura completa prevista, propiedad de bases y comunicaciones.
- contenedores-actuales.mmd: componentes que inicia compose.yaml en el estado actual.

Los PNG del mismo nombre se generan directamente desde las fuentes Mermaid. No editar los PNG por separado. La vista prevista agrupa las dos instancias Solar en un mismo nodo, porque son réplicas del mismo servicio y comparten exclusivamente solar_db. El indexador pertenece a Electrodomésticos; la flecha RabbitMQ → Electrodomésticos representa su consumidor de cambios, no una escritura del broker en la base.

Para regenerar usando Mermaid CLI 12.0.0, instalada en un entorno de herramientas de documentación:

```sh
mmdc -i docs/diagrams/contexto.mmd -o docs/diagrams/contexto.png -b white --size 2400 -s 2 -c docs/diagrams/mermaid.config.json
mmdc -i docs/diagrams/contenedores.mmd -o docs/diagrams/contenedores.png -b white --size 2400 -s 2 -c docs/diagrams/mermaid.config.json
mmdc -i docs/diagrams/contenedores-actuales.mmd -o docs/diagrams/contenedores-actuales.png -b white --size 2400 -s 2 -c docs/diagrams/mermaid.config.json
```

La generación de documentación es independiente del arranque de la aplicación. No se agrega Mermaid a sus dependencias de ejecución. Mermaid CLI valida y renderiza las fuentes; un Chromium compatible debe estar disponible en el entorno de herramientas. Referencia: [Mermaid CLI](https://github.com/mermaid-js/mermaid-cli).

El dibujo de contenedores previstos omite algunas flechas REST para facilitar lectura: Consumo consulta Electrodomésticos; Solar consulta Consumo y Usuarios; Electrodomésticos consulta Usuarios. La sección de comunicaciones de ARCHITECTURE describe esas dependencias. El invitado no produce escrituras/eventos de consumo o estudios; los cambios de catálogo sí producen eventos. Healthchecks internos autenticados y telemetría están previstos en la operación. La vista actual no dibuja conexiones entre servidores Go porque aún no existen; las líneas invisibles del Mermaid solo fijan su orden visual. Con credencial válida, Go responde live 200 y ready 503; negocio 501.
