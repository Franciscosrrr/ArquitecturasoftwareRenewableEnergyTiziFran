# Despliegue de la capacidad M2M

Esta rama contiene un snapshot completo de la implementación verificada para que su código y CI sean coherentes. Render construye únicamente la unidad M2M definida en infra/cloud.Dockerfile; no despliega la web completa, bases, broker ni observabilidad. main y el índice de trabajo original no se modifican por esta rama.

Configurar Web Service con runtime Docker, Dockerfile infra/cloud.Dockerfile, contexto raíz y plan Free. Cuatro secretos de entorno: HEALTHCHECK_TOKEN, INTERNAL_TOKEN, M2M_API_KEY y M2M_DENIED_KEY. Nunca publicarlos en Git, capturas ni frontend. Se conserva el contrato 1.0.1 y el cálculo Go. No reutilizar el origen de la PC como prueba de alojamiento cloud.

Antes de presentar, verificar la URL realmente asignada y distinguir cold start de la llamada caliente. El servicio gratuito puede suspenderse por inactividad; no se promete el presupuesto del consumidor en frío. La prueba desde el backend consumidor queda a cargo de Clínica. Después del despliegue corresponde revisar la integración de los cambios completos en main y comprobar CI.
