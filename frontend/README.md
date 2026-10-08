# frontend

Estado: estructura documental de entrega 1; no contiene implementación ejecutable.

## Responsabilidad

Calculadora accesible sin cuenta, resultados temporales de consumo y paneles, registro/login opcionales, Mis consumos, lugares, edición, historial y administración.

## Organización

React con JavaScript, HTML y CSS; build dentro de Docker.

## Dependencias

Gateway HTTP.

Ver [estructura general](../docs/STRUCTURE.md). Las dependencias y el código se construirán dentro de Docker. No se declara un módulo, una imagen ni una API ya funcionando.

## Borrador y guardado

El borrador invitado vive solo en estado React; no se guarda en localStorage, sessionStorage, IndexedDB ni cookies. Recargar/cerrar lo descarta. Login/registro se muestran sin desmontar la calculadora, para conservar el borrador en memoria. Guardar exige sesión y confirmación explícita; cerrar sesión limpia datos privados y temporales. La interfaz avisa que los resultados invitados no se conservan. No se entrega una API key M2M al navegador.
