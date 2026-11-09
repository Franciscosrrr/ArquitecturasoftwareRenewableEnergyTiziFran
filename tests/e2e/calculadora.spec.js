import {test,expect} from '@playwright/test';
// Pace independent scenarios within the real 1 request/s policy; no mocks or exemptions.
test.beforeEach(async({page})=>{await new Promise(resolve=>setTimeout(resolve,2000));await page.goto('/');await expect(page.getByLabel('Agregar un equipo')).toBeEnabled();await expect(page.getByLabel('Agregar un equipo').locator('option[value="iluminacion"]')).toHaveCount(1);});
test('catálogo real, cálculo mixto y desglose',async({page},testInfo)=>{
 await page.getByLabel('Agregar un equipo').selectOption('iluminacion');
 await page.locator('article').filter({hasText:'Lámpara LED'}).getByLabel('Cantidad',{exact:true}).fill('4');
 await page.getByLabel('Agregar un equipo').selectOption('heladera');
 await page.locator('article').filter({hasText:'Heladera'}).getByLabel('Horas por día').fill('24');
 await page.getByLabel('Agregar un equipo').selectOption('lavarropas');
 await page.getByRole('button',{name:'Calcular consumo ↗'}).click();
 await expect(page.locator('.totals')).toContainText('47,4');
 await expect(page.locator('.totals')).toContainText('1,58');
 await expect(page.getByRole('table')).toContainText('32,4');
 await expect(page.getByRole('alert')).toHaveCount(0);
 await page.screenshot({path:`/results/calculadora-${testInfo.project.name}.png`,fullPage:true});
});
test('editar invalida el resultado y recargar pierde el borrador',async({page})=>{
 await page.getByLabel('Agregar un equipo').selectOption('iluminacion');
 await page.getByRole('button',{name:'Calcular consumo ↗'}).click();
 await expect(page.locator('.totals')).toContainText('1,5');
 await page.getByLabel('Cantidad',{exact:true}).fill('2');
 await expect(page.locator('.results')).toHaveCount(0);
 await page.getByRole('button',{name:'Calcular consumo ↗'}).click();
 await expect(page.locator('.totals')).toContainText('3');
 await page.reload();
 await expect(page.getByText('Agregá tu primer equipo para empezar.')).toBeVisible();
 await expect(page.getByRole('button',{name:'Calcular consumo ↗'})).toBeDisabled();
 await expect(page.locator('article')).toHaveCount(0);
});
test('validación del formulario, quitar equipo y acceso sin sesión',async({page,request})=>{
 await page.getByLabel('Agregar un equipo').selectOption('iluminacion');
 await page.getByLabel('Cantidad',{exact:true}).fill('0');
 await page.getByRole('button',{name:'Calcular consumo ↗'}).click();
 await expect(page.locator('form.workspace input:invalid')).toHaveCount(1);
 await expect(page.locator('.results')).toHaveCount(0);
 await page.getByRole('button',{name:'Quitar Lámpara LED de referencia'}).click();
 await expect(page.getByRole('button',{name:'Calcular consumo ↗'})).toBeDisabled();
 const privateResponse=await request.get('/api/v1/configuraciones');expect(privateResponse.status()).toBe(401);
 const m2mResponse=await request.post('/api/v1/estimaciones-consumo',{data:{diasPeriodo:30,equipos:[]}});expect(m2mResponse.status()).toBe(401);
});
