import {test, expect} from '@playwright/test';

/**
 * Calcula un escenario y verifica que el stock del vendedor afecta la oferta.
 *
 * @method solarQuoteFlow
 * @param {Object} context - Página de prueba de Playwright.
 * @param {import('@playwright/test').Page} context.page - Calculadora abierta.
 * @returns {Promise<void>} Comprueba oferta simulada y conservación del consumo.
 */
async function solarQuoteFlow({page}) {
  await new Promise(resolve=>setTimeout(resolve,3000));
  await page.goto('/');
  await page.getByLabel('Agregar un equipo').selectOption('iluminacion');
  await page.getByRole('button', {name:'Calcular consumo'}).click();
  await expect(page.locator('caption')).toBeVisible();
  await page.getByRole('button', {name:'Cotizar paneles', exact:true}).click();
  await expect(page.getByText('Oferta simulada: stock y precios didácticos.')).toBeVisible();
  await expect(page.getByText(/Stock informado: 1000/)).toBeVisible();
  await page.route('**/api/v1/publico/solar/cotizaciones', route => route.fulfill({
    status:503, contentType:'application/json',
    body:JSON.stringify({codigo:'PROVEEDOR_NO_DISPONIBLE',mensaje:'Vendedor temporalmente no disponible.',traceId:'00000000000000000000000000000000',detalles:[]}),
  }));
  await page.getByRole('button', {name:'Cotizar paneles', exact:true}).click();
  await expect(page.getByRole('alert')).toContainText('Vendedor temporalmente no disponible.');
  await expect(page.locator('caption')).toBeVisible();
  await expect(page.getByText('Oferta simulada: stock y precios didácticos.')).toHaveCount(0);
}

test('cotización comercial y consumo conservado ante fallo', solarQuoteFlow);
