import {test,expect} from '@playwright/test';
/**
 * Comprueba búsqueda solar y que las referencias técnicas aparecen solo al abrir la ficha.
 *
 * @method solarPanelSearch
 * @param {Object} context - Página real del gateway.
 * @param {Object} testInfo - Proyecto de escritorio o móvil y ubicación de evidencia.
 * @returns {Promise<void>} Verifica filtros, resultado y ficha sin interceptar solicitudes.
 */
async function solarPanelSearch({page},testInfo){
 await new Promise(resolve=>setTimeout(resolve,2500));
 await page.goto('/');
 await expect(page.getByText('Buscar y filtrar catálogo',{exact:true})).toHaveCount(0);
 const catalog=page.getByRole('region',{name:'Catálogo de paneles solares'});
 await catalog.locator('summary').filter({hasText:'Buscar y filtrar paneles solares'}).click();
 await catalog.getByLabel('Modelo o descripción',{exact:true}).fill('referencia');
 await catalog.getByLabel('Potencia mínima (W)',{exact:true}).fill('500');
 await catalog.getByLabel('Potencia máxima (W)',{exact:true}).fill('600');
 await catalog.getByLabel('Superficie máxima por panel (m²)',{exact:true}).fill('3');
 await catalog.getByLabel('Precio máximo por panel',{exact:true}).fill('400000');
 await catalog.getByRole('button',{name:'Buscar paneles',exact:true}).click();
 const model=catalog.locator('.panel-card').filter({has:page.getByRole('heading',{name:'Panel de referencia 550 W',exact:true})});
 await expect(model).toBeVisible();
 await expect(model.getByText('Identificador: PANEL-SIM-550',{exact:true})).not.toBeVisible();
 await expect(model.getByRole('heading')).not.toContainText(/\bv\d+\b/);
 await model.locator('summary').click();
 await expect(model.getByText('Identificador: PANEL-SIM-550',{exact:true})).toBeVisible();
 await expect(model).toContainText('Versión de ficha:');
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
 await page.screenshot({path:`/results/paneles-${testInfo.project.name}.png`,fullPage:true});
 await catalog.getByLabel('Modelo o descripción',{exact:true}).fill('inexistente'+Date.now());
 await catalog.getByRole('button',{name:'Buscar paneles',exact:true}).click();
 await expect(catalog.getByRole('status')).toContainText('0 modelos encontrados');
}
test('paneles solares: filtros y ficha técnica opcional',solarPanelSearch);
