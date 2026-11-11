# Cierre de la entrega — comisión A1

Integrantes: **Tiziano Nicolas, 2407889**, y **Francisco Skrie, 2423159**. JTP informado por el equipo: **Nicolás Rodríguez**. Proyecto: Renewable Energy.

## Repositorio y versión evaluable

Repositorio público: https://github.com/Franciscosrrr/ArquitecturasoftwareRenewableEnergyTiziFran. La versión evaluable debe quedar en `main`. Las ramas de trabajo se revisan antes de integrarse; la rama usada para el despliegue M2M anterior no sustituye la entrega de la aplicación completa.

Esta revisión documental y la integración real se guardaron en el árbol local. Antes de subir: revisar `git status` y los cambios, conservar fuera de Git `.env`/`.runtime`, incorporar los archivos del proyecto y comprobar CI. No usar un hash de otro despliegue como si correspondiera a la entrega final.

El flujo definido utiliza ramas cortas desde main para cambios concretos, revisión del otro integrante e integración tras los checks de CI. La publicación M2M utilizó `deploy/estimaciones-consumo-20261109`; ese despliegue independiente se conserva con su commit registrado. El repositorio incluye CI de verificación, pero esta documentación no afirma que existan reglas de protección de rama ni revisiones obligatorias configuradas en GitHub. Revisar el diff final entre ambos antes de publicar la entrega.

Después de publicar la versión definitiva:

```powershell
git fetch origin
git rev-parse origin/main
```

Copiar ese hash en el Word. Si luego se publica otro commit, volver a actualizarlo. No se afirma haber realizado commit, push o revisión de CI de esta auditoría.

## Ficha de presentación Word

Completar proyecto, comisión/grupo e integrantes, JTP, resumen del dominio, URL del repositorio y hash final de main. La capacidad propia es `POST https://renewable-consumo-v1.onrender.com/api/v1/estimaciones-consumo`, consumida por **Portal Paciente Clínica**. Estima kWh; Clínica aplica su tarifa para calcular pesos.

La capacidad consumida pertenece a **Comparador de Proveedores**: cotización y disponibilidad de paneles por SKU/cantidad, utilizada en Solar para seleccionar una alternativa con stock por su costo total. Ahora está integrada; no debe figurar «proveedor real pendiente». Registrar también que las fallas técnicas activan un respaldo simulado identificado.

Según la ficha recibida, se entrega un único documento por equipo y se eliminan las filas sobrantes. Solo se adjunta ZIP adicional si hay documentación de la entrega que no esté incluida en GitHub. La presentación y el guion preparados por separado deben subirse al repositorio o adjuntarse si forman parte de ese material.

## Disponibilidad durante la evaluación

Clínica utiliza Render y sus claves privadas; su API puede reactivarse lentamente y no depende de la PC. La web completa requiere PC, Docker y red. Consultar la URL observada del túnel con `scripts/publicar-web.ps1 estado`; puede cambiar al reiniciarse. Mantener los volúmenes y `.env`; `scripts/compose.ps1 stop` conserva datos. No utilizar `down -v` para preparar la exposición.

[Índice documental](README.md), [pruebas actuales](VERIFICACION-FINAL.md), [configuración](CONFIGURACION.md) y [limitaciones](ARCHITECTURE.md).
