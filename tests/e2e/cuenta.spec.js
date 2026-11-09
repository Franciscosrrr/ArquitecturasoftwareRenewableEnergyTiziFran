import {test,expect} from '@playwright/test';
/**
 * Comprueba promoción explícita del borrador, evaluación durable e historial desde el navegador.
 *
 * @method durableAccountFlow
 * @param {Object} context - Página real conectada al gateway.
 * @param {Object} testInfo - Proyecto de navegador y ubicación de evidencias.
 * @returns {Promise<void>} Verifica cuenta, restauración y estudio sin interceptar HTTP.
 */
async function durableAccountFlow({page},testInfo){
 test.setTimeout(60000);
 await new Promise(resolve=>setTimeout(resolve,5000));
 await page.goto('/');await page.getByLabel('Agregar un equipo').selectOption('iluminacion');
 const account=page.locator('.account');
 await account.getByLabel('Email',{exact:true}).fill('navegador-'+Date.now()+'-'+testInfo.project.name+'@example.test');
 await account.getByLabel('Contraseña (mínimo 12 caracteres)',{exact:true}).fill('contraseña-de-navegador-segura');
 await account.getByRole('button',{name:'Crear cuenta',exact:true}).click();
 await account.getByRole('button',{name:'Cerrar sesión',exact:true}).waitFor();
 await account.locator('summary').filter({hasText:/^Crear lugar$/}).click();
 await account.getByLabel('Nombre del lugar',{exact:true}).fill('Hospital demo');
 await account.getByLabel('Ubicación',{exact:true}).fill('Supuesto manual');
 await account.getByLabel('Superficie útil (m², opcional)',{exact:true}).fill('40');
 await account.getByRole('button',{name:'Crear lugar',exact:true}).click();
 await account.getByRole('button',{name:'Guardar consumo',exact:true}).click();
 const saved=account.locator('article.saved').filter({has:page.getByRole('button',{name:'Calcular paneles',exact:true})});
 await expect(saved.getByRole('heading',{name:'Mi consumo',exact:true})).toBeVisible();
 await expect(saved.getByRole('heading')).not.toContainText(/v[0-9]+/);
 await saved.getByRole('button',{name:'Archivar',exact:true}).click();
 await expect(saved).toHaveCount(0);
 await account.getByLabel('Mostrar consumos archivados',{exact:true}).check();
 await saved.getByRole('button',{name:'Restaurar',exact:true}).click();
 await saved.locator('summary').filter({hasText:'Ver ficha del consumo'}).click();
 await expect(saved).toContainText('Versión de contenido: 1');
 await saved.locator('summary').filter({hasText:'Ver ficha del consumo'}).click();
 const location=account.locator('.place-detail');
 await location.locator('summary').first().click();
 await location.getByRole('button',{name:'Archivar lugar',exact:true}).click();
 await expect(account.getByRole('combobox',{name:'Lugar',exact:true}).locator('option:checked')).toHaveText('Hospital demo (Archivado)');
 await expect(account.getByRole('button',{name:'Guardar cambios',exact:true})).toBeDisabled();
 await expect(saved.getByRole('button',{name:'Calcular paneles',exact:true})).toBeDisabled();
 await location.getByRole('button',{name:'Restaurar lugar',exact:true}).click();
 await expect(account.getByRole('combobox',{name:'Lugar',exact:true}).locator('option:checked')).toHaveText('Hospital demo');
 await location.locator('summary').first().click();
 await saved.getByRole('button',{name:'Calcular paneles',exact:true}).click();
 await expect(account.getByRole('status')).toContainText('Estamos calculando tus paneles');
 const evaluation=account.locator('details').filter({has:page.locator('summary').filter({hasText:/configuración v1/})}).first();
 await evaluation.locator('summary').first().click();
 await expect(evaluation).toContainText('Listo',{timeout:25000});
 await expect(evaluation).toContainText('Oferta del vendedor simulado');
 await saved.getByRole('button',{name:'Abrir para editar',exact:true}).click();
 await page.locator('.equipment').getByLabel('Cantidad',{exact:true}).fill('2');
 await account.getByRole('button',{name:'Guardar cambios',exact:true}).click();
 await saved.locator('summary').filter({hasText:'Ver ficha del consumo'}).click();
 await expect(saved).toContainText('Versión de contenido: 2');
 await saved.locator('summary').filter({hasText:'Ver ficha del consumo'}).click();
 await expect(evaluation).toContainText('Versión anterior');
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
 await page.screenshot({path:`/results/cuenta-${testInfo.project.name}.png`,fullPage:true});
 await page.reload();await account.getByRole('button',{name:'Cerrar sesión',exact:true}).waitFor();
 await account.getByRole('button',{name:'Abrir para editar',exact:true}).click();
 await expect(page.locator('.equipment').getByLabel('Cantidad',{exact:true})).toHaveValue('2');
 await expect(page.getByRole('button',{name:'Calcular consumo ↗'})).toBeEnabled();
 await page.once('dialog',dialog=>dialog.accept());
 await account.getByRole('button',{name:'Eliminar consumo',exact:true}).click();
 await expect(saved).toHaveCount(0);
 await expect(account).toContainText('Consumo eliminado · configuración v1');
 const place=account.locator('.place-detail');
 await place.locator('summary').first().click();
 await page.once('dialog',dialog=>dialog.accept());
 await place.getByRole('button',{name:'Eliminar lugar',exact:true}).click();
 await expect(place).toHaveCount(0);
 await account.getByRole('button',{name:'Cerrar sesión',exact:true}).click();
 await expect(account.getByRole('button',{name:'Iniciar sesión',exact:true})).toBeVisible();
}
test('cuenta, guardado, estudio e historial inmutable',durableAccountFlow);
