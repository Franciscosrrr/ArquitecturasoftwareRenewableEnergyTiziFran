package transport

import (
	"context"
	"renewable.local/solar/internal/adapters/search"
	"renewable.local/solar/internal/application"
	"renewable.local/solar/internal/domain"
)

/**
 * Enriquece fichas con una sola procedencia comercial por consulta.
 */
type panelPricer interface {
	/**
	 * Enriquece fichas técnicas con precio, stock y procedencia comercial.
	 *
	 * @method PricePanels
	 * @param {context.Context} ctx - Presupuesto cancelable de consulta.
	 * @param {[]domain.Panel} panels - Fichas técnicas a enriquecer con ofertas.
	 * @returns {[]domain.Panel, error} Fichas y oferta coherentes o error de dependencia.
	 */
	PricePanels(context.Context, []domain.Panel) ([]domain.Panel, error)
}

/**
 * Busca en el motor por especificaciones y aplica precio y paginación a las ofertas actuales.
 *
 * @method pricedCatalog
 * @param {context.Context} ctx - Presupuesto de buscador y proveedor.
 * @param {*search.Runtime} runtime - Motor existente, con respaldo Mongo.
 * @param {search.Query} query - Filtros y página solicitados.
 * @param {[]application.ModelSupplier} suppliers - Fuente comercial configurada.
 * @returns {search.Page, string, error} Página coherente con los precios mostrados.
 */
func pricedCatalog(ctx context.Context, runtime *search.Runtime, query search.Query, suppliers []application.ModelSupplier) (search.Page, string, error) {
	if len(suppliers) == 0 {
		return runtime.List(ctx, query)
	}
	pricer, ok := suppliers[0].(panelPricer)
	if !ok {
		return runtime.List(ctx, query)
	}
	technical := query
	technical.MaxCents = nil
	technical.Order = "nombre"
	technical.Page = 1
	technical.Size = 50
	panels := []domain.Panel{}
	source := ""
	for {
		page, origin, err := runtime.List(ctx, technical)
		if err != nil {
			return search.Page{}, origin, err
		}
		source = origin
		panels = append(panels, page.Items...)
		if len(panels) >= page.Total || len(page.Items) == 0 {
			break
		}
		if len(panels) >= 1000 {
			return search.Page{}, source, domain.ErrUnavailable
		}
		technical.Page++
	}
	priced, err := pricer.PricePanels(ctx, panels)
	if err != nil {
		return search.Page{}, source, err
	}
	return search.Fallback(query, priced), source, nil
}
