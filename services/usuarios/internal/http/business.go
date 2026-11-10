package transport

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"renewable.local/platform"
	"renewable.local/usuarios/internal/identity"
	"strings"
)

type Config struct {
	Store         *identity.Store
	InternalToken string
	SecureCookie  bool
}

/**
 * Conecta rutas HTTP, salud y dependencias de negocio del servicio.
 *
 * @method NewHandler
 * @param {string} token - Credencial técnica del servicio.
 * @param {...Config} configs - Valor de configs requerido por la operación.
 * @returns {http.Handler} Resultado de la operación; error si no puede completarse.
 */
func NewHandler(token string, configs ...Config) http.Handler {
	if len(configs) == 0 {
		return legacyHandler(token)
	}
	cfg := configs[0]
	r := platform.Router()
	platform.Health(r, token, cfg.Store.DB.PingContext)
	limiter := platform.NewLimiter()
	passwordSlots := make(chan struct{}, 4)
	fail := func(c *gin.Context, err error) {
		status, code := 503, "SERVICIO_NO_DISPONIBLE"
		if errors.Is(err, identity.ErrCredentials) {
			status, code = 401, "NO_AUTENTICADO"
		}
		if errors.Is(err, identity.ErrValidation) {
			status, code = 422, "VALIDACION"
		}
		if errors.Is(err, identity.ErrConflict) {
			status, code = 409, "CONFLICTO"
		}
		platform.Error(c, status, code, "No se pudo completar la operación de identidad.", nil)
	}
	cookie := func(c *gin.Context, value string, age int) {
		http.SetCookie(c.Writer, &http.Cookie{Name: "renewable_refresh", Value: value, Path: "/api/v1/auth", HttpOnly: true, Secure: cfg.SecureCookie, SameSite: http.SameSiteStrictMode, MaxAge: age})
	}
	auth := r.Group("/api/v1/auth")
	auth.Use(func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" {
			u, err := url.Parse(origin)
			host := c.GetHeader("X-Forwarded-Host")
			if host == "" {
				host = c.Request.Host
			}
			if err != nil || u.Host != host {
				platform.Error(c, 403, "SIN_PERMISO", "Origen no permitido.", nil)
				c.Abort()
				return
			}
		}
		if !limiter.Allow(c.ClientIP()) {
			c.Header("Retry-After", "1")
			platform.Error(c, 429, "LIMITE_DE_TRAFICO", "Reintente más tarde.", nil)
			c.Abort()
			return
		}
		c.Next()
	})
	auth.POST("/registro", func(c *gin.Context) {
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !platform.Decode(c, &input) {
			return
		}
		select {
		case passwordSlots <- struct{}{}:
			defer func() { <-passwordSlots }()
		default:
			platform.Error(c, 429, "LIMITE_DE_TRAFICO", "Intente más tarde.", nil)
			return
		}
		user, err := cfg.Store.Create(c.Request.Context(), input.Email, input.Password, "USUARIO")
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(201, user)
	})
	auth.POST("/login", func(c *gin.Context) {
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !platform.Decode(c, &input) {
			return
		}
		select {
		case passwordSlots <- struct{}{}:
			defer func() { <-passwordSlots }()
		default:
			platform.Error(c, 429, "LIMITE_DE_TRAFICO", "Intente más tarde.", nil)
			return
		}
		session, err := cfg.Store.Login(c.Request.Context(), input.Email, input.Password)
		if err != nil {
			fail(c, err)
			return
		}
		cookie(c, session.Refresh, 7*86400)
		c.JSON(200, session)
	})
	auth.POST("/refresh", func(c *gin.Context) {
		value, err := c.Cookie("renewable_refresh")
		if err != nil {
			fail(c, identity.ErrCredentials)
			return
		}
		session, err := cfg.Store.Refresh(c.Request.Context(), value)
		if err != nil {
			cookie(c, "", -1)
			fail(c, err)
			return
		}
		cookie(c, session.Refresh, 7*86400)
		c.JSON(200, session)
	})
	auth.POST("/logout", func(c *gin.Context) {
		value, _ := c.Cookie("renewable_refresh")
		if err := cfg.Store.Logout(c.Request.Context(), value); err != nil {
			fail(c, err)
			return
		}
		cookie(c, "", -1)
		c.Status(204)
	})
	r.POST("/internal/v1/identidad/validar", func(c *gin.Context) {
		if !platform.CredentialOK(c.GetHeader("Authorization"), "Bearer "+cfg.InternalToken) {
			c.Status(401)
			return
		}
		var input struct {
			Token string `json:"token"`
		}
		if !platform.Decode(c, &input) {
			return
		}
		user, err := cfg.Store.Validate(c.Request.Context(), input.Token)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, user)
	})
	r.GET("/api/v1/perfil", platform.Internal(cfg.InternalToken, false), func(c *gin.Context) {
		user, err := cfg.Store.Validate(c.Request.Context(), strings.TrimPrefix(c.GetHeader("X-Access-Token"), "Bearer "))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, user)
	})
	admin := r.Group("/api/v1/admin/usuarios", platform.Internal(cfg.InternalToken, true))
	admin.GET("", func(c *gin.Context) {
		users, err := cfg.Store.Users(c.Request.Context())
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, users)
	})
	admin.PUT("/:id/rol", func(c *gin.Context) {
		var input struct {
			Role string `json:"rol"`
		}
		if !platform.Decode(c, &input) {
			return
		}
		if err := cfg.Store.Role(c.Request.Context(), c.Param("id"), input.Role); err != nil {
			fail(c, err)
			return
		}
		c.Status(204)
	})
	return platform.Instrument("usuarios", r)
}
