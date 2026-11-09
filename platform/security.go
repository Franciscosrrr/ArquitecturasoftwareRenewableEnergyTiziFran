package platform

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

type Actor struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"rol"`
}

/**
 * Genera un UUID aleatorio para una nueva entidad de negocio.
 *
 * @method RandomID
 * @returns {string} Resultado de la operación; error si no puede completarse.
 */
func RandomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("no hay fuente aleatoria")
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return hex.EncodeToString(b[:4]) + "-" + hex.EncodeToString(b[4:6]) + "-" + hex.EncodeToString(b[6:8]) + "-" + hex.EncodeToString(b[8:10]) + "-" + hex.EncodeToString(b[10:])
}

/**
 * Genera una credencial aleatoria sin imprimirla.
 *
 * @method RandomSecret
 * @returns {string} Resultado de la operación; error si no puede completarse.
 */
func RandomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("no hay fuente aleatoria")
	}
	return hex.EncodeToString(b)
}

/**
 * Calcula SHA-256 para comparar contenido o guardar credenciales derivadas.
 *
 * @method Digest
 * @param {string} s - Valor de s requerido por la operación.
 * @returns {string} Resultado de la operación; error si no puede completarse.
 */
func Digest(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }

/**
 * Compara una credencial técnica en tiempo constante y rechaza configuraciones débiles.
 *
 * @method CredentialOK
 * @param {string} actual - Valor de actual requerido por la operación.
 * @param {string} expected - Valor de expected requerido por la operación.
 * @returns {bool} Resultado de la operación; error si no puede completarse.
 */
func CredentialOK(actual, expected string) bool {
	a := sha256.Sum256([]byte(actual))
	b := sha256.Sum256([]byte(expected))
	return len(expected) >= 32 && subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

/**
 * Decodifica un único objeto JSON acotado y rechaza campos desconocidos.
 *
 * @method Decode
 * @param {*gin.Context} c - Solicitud HTTP en curso.
 * @param {any} v - Valor de v requerido por la operación.
 * @returns {bool} Resultado de la operación; error si no puede completarse.
 */
func Decode(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 262144)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	d.UseNumber()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		Error(c, 422, "VALIDACION", "Cuerpo JSON inválido o campos desconocidos.", nil)
		return false
	}
	return true
}

/**
 * Exige el secreto interno y el actor autenticado, con permiso administrativo opcional.
 *
 * @method Internal
 * @param {string} token - Credencial técnica del servicio.
 * @param {bool} admin - Valor de admin requerido por la operación.
 * @returns {gin.HandlerFunc} Resultado de la operación; error si no puede completarse.
 */
func Internal(token string, admin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CredentialOK(c.GetHeader("Authorization"), "Bearer "+token) || c.GetHeader("X-Actor-ID") == "" {
			Error(c, 401, "NO_AUTENTICADO", "Se requiere una sesión válida.", nil)
			c.Abort()
			return
		}
		if admin && c.GetHeader("X-Actor-Role") != "ADMINISTRADOR" {
			Error(c, 403, "SIN_PERMISO", "Se requiere permiso administrativo.", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

/**
 * Recupera el propietario previamente validado por el gateway.
 *
 * @method ActorID
 * @param {*gin.Context} c - Solicitud HTTP en curso.
 * @returns {string} Resultado de la operación; error si no puede completarse.
 */
func ActorID(c *gin.Context) string { return c.GetHeader("X-Actor-ID") }
