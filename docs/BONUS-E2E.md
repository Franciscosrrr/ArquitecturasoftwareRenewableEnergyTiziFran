# Bonus anunciado en entrega 2: tests e2e

El equipo anuncia un único bonus: creación de tests end-to-end. Se incluyen pruebas de navegador con Playwright/Chromium para escritorio y móvil, junto con tests HTTP contra el sistema real. No se propone otro bonus.

## Recorrido cubierto

El navegador descarga React desde el gateway, consulta Electrodomésticos/MySQL, agrega equipos, completa hábitos, envía un cálculo a Consumo y verifica el resultado renderizado. No se interceptan respuestas ni se sustituye la base por un mock.

1. Escenario mixto con iluminación, heladera y lavarropas: 47,4 kWh y 1,58 kWh/día, con desglose.
2. Editar invalida el resultado; recalcular usa el nuevo valor; recargar pierde el borrador y deshabilita calcular sin equipos.
3. Cantidad cero activa la validación del formulario; quitar equipo deja el escenario vacío; las rutas privadas y M2M rechazan el acceso sin credenciales.

Tres casos × dos tamaños de pantalla = seis pruebas de navegador. Los tests HTTP agregan esquema de respuesta, autenticación M2M 401/403, catálogo real, búsqueda/paginación, cuerpo máximo, Content-Type, JSON inválido y salud autenticada.

## Ejecución

```powershell
.\scripts\compose.ps1 up -d --build
.\scripts\compose.ps1 --profile verificar run --build --rm verificar-e2e
```

En Linux/macOS usar `sh scripts/compose.sh`. Se utiliza la imagen oficial `mcr.microsoft.com/playwright:v1.64.1-noble` y `@playwright/test` 1.64.1, con un solo worker. Los archivos quedan en `docs/evidence/entrega2/e2e`: reporte JSON, capturas y, ante fallos, trazas/adjuntos. La CI ejecuta el mismo recorrido y conserva los reportes.

La aprobación y el puntaje del bonus corresponden a la cátedra. Los tests no cubren aún identidad, guardados ni Solar, que no forman parte del flujo implementado en este hito.
