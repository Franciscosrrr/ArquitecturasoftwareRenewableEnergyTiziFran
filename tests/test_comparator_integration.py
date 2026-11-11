"""Contrato HTTP de las ofertas reales y del respaldo simulado de Solar."""
import json
import os
import unittest
from pathlib import Path
from decimal import Decimal
from test_real import call, CATALOG_REQUEST
from mock.server import validate

ROOT=Path(__file__).resolve().parents[1]
SOLAR=json.loads((ROOT/'docs/contracts/solar-v1/openapi.json').read_text(encoding='utf-8-sig'))
SEARCH=json.loads((ROOT/'docs/contracts/panel-search-v1/openapi.json').read_text(encoding='utf-8-sig'))

class CommercialIntegration(unittest.TestCase):
    """/**
     * Verifica procedencia, contrato y filtros con el backend y sus dependencias reales.
     *
     * @method CommercialIntegration
     * @returns {CommercialIntegration} Casos repetibles en modo real o de falla controlada.
     */
    """
    def test_quote_source_and_commercial_total(self):
        """/**
         * Comprueba el origen efectivo y el total con descuento y flete sin exponer claves.
         *
         * @method test_quote_source_and_commercial_total
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        status,result,_=call('/api/v1/publico/solar/cotizaciones',{'consumo':CATALOG_REQUEST,'horasSolPico':4,'factorRendimiento':.8})
        self.assertEqual(status,200,result)
        quote=result['cotizacion'];validate(quote,SOLAR['components']['schemas']['Cotizacion'],SOLAR)
        self.assertIn(quote['origen'],['externo','simulado'])
        self.assertEqual(result['modoProveedor'],quote['origen'])
        if os.getenv('EXPECTED_PROVIDER_SOURCE'):self.assertEqual(quote['origen'],os.environ['EXPECTED_PROVIDER_SOURCE'])
        if quote['origen']=='externo':
            self.assertEqual(quote['cotizacionId'],'');self.assertTrue(quote['referenciaProveedor'])
            unit=Decimal(quote['precioUnitarioCentavos']);amount=unit*quote['cantidad']
            if quote.get('descuentoAplicado'):amount*=1-Decimal(quote['descuentoPorcentaje'])/100
            amount+=quote.get('fleteCentavos',0)
            self.assertEqual(Decimal(quote['totalCentavos']),amount.quantize(Decimal('1'),rounding='ROUND_HALF_UP'))
        elif os.getenv('EXPECTED_PROVIDER_SOURCE')=='simulado':self.assertEqual(quote.get('motivoSimulacion'),'PROVEEDOR_NO_DISPONIBLE')

    def test_catalog_source_and_price_filter(self):
        """/**
         * Filtra con el precio que se muestra y conserva etiquetas explícitas al degradar.
         *
         * @method test_catalog_source_and_price_filter
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        status,page,_=call('/api/v1/publico/paneles/busqueda?moneda=ARS&orden=precio')
        self.assertEqual(status,200,page);validate(page,SEARCH['components']['schemas']['PaginaPaneles'],SEARCH)
        self.assertGreaterEqual(page['total'],2)
        quoted=[p for p in page['items'] if p.get('oferta')]
        self.assertGreaterEqual(len(quoted),2)
        for panel in quoted:
            self.assertEqual(panel['precioCentavos'],panel['oferta']['precioUnitarioCentavos'])
            self.assertEqual(panel['origenOferta'],panel['oferta']['origen'])
            if os.getenv('EXPECTED_PROVIDER_SOURCE'):self.assertEqual(panel['origenOferta'],os.environ['EXPECTED_PROVIDER_SOURCE'])
        ceiling=min(p['precioCentavos'] for p in quoted)
        pesos=format(Decimal(ceiling)/100,'f')
        status,filtered,_=call('/api/v1/publico/paneles/busqueda?moneda=ARS&orden=precio&precioMax='+pesos)
        self.assertEqual(status,200,filtered);self.assertTrue(filtered['items'])
        self.assertTrue(all(0<p['precioCentavos']<=ceiling for p in filtered['items']))

if __name__=='__main__':unittest.main(verbosity=2)
