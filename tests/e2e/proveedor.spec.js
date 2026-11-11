import {test,expect} from '@playwright/test';

/**
 * Comprueba que la oferta y el catálogo muestran su procedencia real o simulada desde el backend.
 *
 * @method providerOriginFlow
 * @param {Object} context - Página conectada al gateway real.
 * @param {Object} testInfo - Variante de pantalla y destino de evidencias.
 * @returns {Promise<void>} Verifica etiquetas, flete, contrato visible y ajuste móvil.
 */
async function providerOriginFlow({page},testInfo){
  const expected=process.env.EXPECTED_PROVIDER_SOURCE;
  await page.goto('/');
  await page.getByLabel('Agregar un equipo').selectOption('iluminacion');
  await page.getByRole('button',{name:/^Calcular consumo/}).click();
  const pending=page.waitForResponse(response=>response.url().endsWith('/api/v1/publico/solar/cotizaciones'));
  await page.getByRole('button',{name:'Cotizar paneles',exact:true}).click();
  const response=await pending;expect(response.status()).toBe(200);
  const body=await response.json();if(expected)expect(body.cotizacion.origen).toBe(expected);
  const offer=page.getByRole('region',{name:'Cotización solar'}).locator('.commercial-offer');
  if(body.cotizacion.origen==='externo'){
    await expect(offer).toContainText('Oferta del proveedor:');await expect(offer).toContainText('Incluye flete:');await expect(offer).not.toContainText('Oferta simulada:');
  }else{
    await expect(offer).toContainText('Oferta simulada: stock y precios didácticos.');
    if(expected==='simulado')await expect(offer).toContainText('El proveedor real no está disponible.');
  }
  const catalog=page.getByRole('region',{name:'Catálogo de paneles solares'});
  await catalog.locator('summary').filter({hasText:'Buscar y filtrar paneles solares'}).click();
  await catalog.getByRole('button',{name:'Buscar paneles',exact:true}).click();
  await expect(catalog.locator('.panel-card')).toHaveCount(2);
  for(const card of await catalog.locator('.panel-card').all()){
    await expect(card).toContainText(body.cotizacion.origen==='externo'?'Precio y stock del proveedor.':'Panel simulado · precio y stock de ejemplo.');
  }
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
  await page.screenshot({path:`/results/proveedor-${body.cotizacion.origen}-${testInfo.project.name}.png`,fullPage:true});
}

test('procedencia comercial real y respaldo simulado visibles',providerOriginFlow);
