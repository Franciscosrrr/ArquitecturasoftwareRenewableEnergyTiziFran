# Gateway - Entrega 2

Sirve React y publica únicamente las rutas de consumo/catálogo implementadas, junto con los contratos JSON. Reenvía con OpenTelemetry y timeout de 8 s. No publica el endpoint interno de resolución. Las rutas privadas no implementadas requieren sesión y no permiten escrituras anónimas. Los healthchecks exigen token operativo.

M2M se autentica en Consumo; el navegador usa rutas públicas sin API key. Los límites por IP/hash M2M se aplican sin confiar en X-Forwarded-For enviado por Internet. JWT y balanceo activo de Solar son diseño pendiente para la final, no capacidades de este gateway inicial.
