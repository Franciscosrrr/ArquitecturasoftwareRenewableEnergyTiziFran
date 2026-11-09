package private

import (
	"github.com/gin-gonic/gin"
	"renewable.local/platform"
)

/**
 * Expone archivo independiente del contenido y eliminación de recursos del propietario.
 *
 * @method registerLifecycle
 * @param {*gin.RouterGroup} api - Grupo con actor autenticado por el gateway.
 * @returns {void} Registra rutas sin modificar operaciones M2M ni snapshots.
 */
func (s Service) registerLifecycle(api *gin.RouterGroup) {
	for _, resource := range []struct {
		path  string
		place bool
	}{{"/lugares", true}, {"/configuraciones", false}} {
		api.PATCH(resource.path+"/:id/archivo", func(c *gin.Context) {
			var input struct {
				Version  int64 `json:"version"`
				Archived *bool `json:"archivado"`
			}
			if !platform.Decode(c, &input) {
				return
			}
			if input.Version < 1 || input.Archived == nil {
				platform.Error(c, 422, "VALIDACION", "Se requiere versión y estado de archivo.", nil)
				return
			}
			if err := s.Store.SetArchived(c.Request.Context(), platform.ActorID(c), c.Param("id"), input.Version, resource.place, *input.Archived); err != nil {
				fail(c, err)
				return
			}
			c.Status(204)
		})
		api.DELETE(resource.path+"/:id", func(c *gin.Context) {
			var input struct {
				Version int64 `json:"version"`
			}
			if !platform.Decode(c, &input) {
				return
			}
			if input.Version < 1 {
				platform.Error(c, 422, "VALIDACION", "Se requiere la versión del recurso que querés eliminar.", nil)
				return
			}
			if err := s.Store.Delete(c.Request.Context(), platform.ActorID(c), c.Param("id"), input.Version, resource.place); err != nil {
				fail(c, err)
				return
			}
			c.Status(204)
		})
	}
}
