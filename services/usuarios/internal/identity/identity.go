package identity

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
	"net/mail"
	"renewable.local/platform"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrCredentials = errors.New("credenciales inválidas")
var ErrConflict = errors.New("recurso duplicado o transición inválida")
var ErrValidation = errors.New("datos inválidos")

type Store struct {
	DB     *sql.DB
	Secret string
}
type Session struct {
	Access  string         `json:"accessToken"`
	User    platform.Actor `json:"usuario"`
	Refresh string         `json:"-"`
}

/**
 * Deriva una contraseña con Argon2id y salt aleatorio independiente por usuario.
 *
 * @method Hash
 * @param {string} password - Valor de password requerido por la operación.
 * @returns {string, error} Resultado de la operación; error si no puede completarse.
 */
func Hash(password string) (string, error) {
	if utf8.RuneCountInString(password) < 12 || len(password) > 512 {
		return "", ErrValidation
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32)
	return "$argon2id$v=19$m=19456,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

/**
 * Verifica el hash Argon2id y rechaza representaciones incompatibles.
 *
 * @method Check
 * @param {string} password - Valor de password requerido por la operación.
 * @param {string} encoded - Valor de encoded requerido por la operación.
 * @returns {bool} Resultado de la operación; error si no puede completarse.
 */
func Check(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=19456,t=2,p=1" || len(password) > 512 {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[4])
	if e != nil || len(salt) != 16 {
		return false
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[5])
	if e != nil || len(expected) != 32 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

/**
 * Normaliza el email y rechaza nombres de presentación o direcciones inválidas.
 *
 * @method Normalize
 * @param {string} email - Valor de email requerido por la operación.
 * @returns {string, error} Resultado de la operación; error si no puede completarse.
 */
func Normalize(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(email) > 254 {
		return "", ErrValidation
	}
	return email, nil
}

/**
 * Crea o migra el esquema propio de manera repetible sin eliminar datos existentes.
 *
 * @method Migrate
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Migrate(ctx context.Context) error {
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS usuarios (id CHAR(36) PRIMARY KEY,email VARCHAR(254) NOT NULL UNIQUE,password_hash VARCHAR(255) NOT NULL,rol ENUM('USUARIO','ADMINISTRADOR') NOT NULL,created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS sesiones (id CHAR(36) PRIMARY KEY,usuario_id CHAR(36) NOT NULL,refresh_hash CHAR(64) NOT NULL UNIQUE,expires_at BIGINT NOT NULL,FOREIGN KEY(usuario_id) REFERENCES usuarios(id),INDEX(usuario_id))`,
	} {
		if _, err := s.DB.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

/**
 * Crea un usuario con email único, rol controlado y contraseña derivada.
 *
 * @method Create
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} email - Valor de email requerido por la operación.
 * @param {string} password - Valor de password requerido por la operación.
 * @param {string} role - Valor de role requerido por la operación.
 * @returns {platform.Actor, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Create(ctx context.Context, email, password, role string) (platform.Actor, error) {
	email, err := Normalize(email)
	if err != nil {
		return platform.Actor{}, err
	}
	hash, err := Hash(password)
	if err != nil {
		return platform.Actor{}, err
	}
	if role != "USUARIO" && role != "ADMINISTRADOR" {
		return platform.Actor{}, ErrValidation
	}
	user := platform.Actor{ID: platform.RandomID(), Email: email, Role: role}
	if _, err = s.DB.ExecContext(ctx, "INSERT INTO usuarios(id,email,password_hash,rol) VALUES(?,?,?,?)", user.ID, email, hash, role); err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			return platform.Actor{}, ErrConflict
		}
		return platform.Actor{}, err
	}
	return user, nil
}

/**
 * Crea el administrador inicial únicamente si todavía no existe ninguno.
 *
 * @method Bootstrap
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} email - Valor de email requerido por la operación.
 * @param {string} password - Valor de password requerido por la operación.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Bootstrap(ctx context.Context, email, password string) error {
	var count int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usuarios WHERE rol='ADMINISTRADOR'").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := s.Create(ctx, email, password, "ADMINISTRADOR")
	return err
}

/**
 * Firma un access token de quince minutos ligado a una sesión revocable.
 *
 * @method token
 * @param {platform.Actor} user - Valor de user requerido por la operación.
 * @param {string} session - Valor de session requerido por la operación.
 * @returns {string, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) token(user platform.Actor, session string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{Issuer: "renewable-usuarios", Subject: user.ID, Audience: jwt.ClaimStrings{"renewable-web"}, ID: session, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute))}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.Secret))
}

/**
 * Verifica credenciales y crea una sesión con refresh almacenado solo como hash.
 *
 * @method Login
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} email - Valor de email requerido por la operación.
 * @param {string} password - Valor de password requerido por la operación.
 * @returns {Session, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Login(ctx context.Context, email, password string) (Session, error) {
	normalized, err := Normalize(email)
	if err != nil {
		return Session{}, ErrCredentials
	}
	var user platform.Actor
	var hash string
	if err = s.DB.QueryRowContext(ctx, "SELECT id,email,rol,password_hash FROM usuarios WHERE email=?", normalized).Scan(&user.ID, &user.Email, &user.Role, &hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrCredentials
		}
		return Session{}, err
	}
	if !Check(password, hash) {
		return Session{}, ErrCredentials
	}
	id, refresh := platform.RandomID(), platform.RandomSecret()
	if _, err = s.DB.ExecContext(ctx, "INSERT INTO sesiones(id,usuario_id,refresh_hash,expires_at) VALUES(?,?,?,?)", id, user.ID, platform.Digest(refresh), time.Now().Add(7*24*time.Hour).Unix()); err != nil {
		return Session{}, err
	}
	access, err := s.token(user, id)
	return Session{access, user, refresh}, err
}

/**
 * Verifica firma, emisor, audiencia, vencimiento y vigencia SQL de la sesión.
 *
 * @method Validate
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} raw - Cuerpo original recibido para validar.
 * @returns {platform.Actor, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Validate(ctx context.Context, raw string) (platform.Actor, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) { return []byte(s.Secret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("renewable-usuarios"), jwt.WithAudience("renewable-web"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return platform.Actor{}, ErrCredentials
	}
	var user platform.Actor
	err = s.DB.QueryRowContext(ctx, "SELECT u.id,u.email,u.rol FROM usuarios u JOIN sesiones s ON s.usuario_id=u.id WHERE s.id=? AND u.id=? AND s.expires_at>?", claims.ID, claims.Subject, time.Now().Unix()).Scan(&user.ID, &user.Email, &user.Role)
	if err != nil {
		return user, ErrCredentials
	}
	return user, nil
}

/**
 * Rota el refresh bajo bloqueo transaccional y conserva la sesión del usuario.
 *
 * @method Refresh
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} raw - Cuerpo original recibido para validar.
 * @returns {Session, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Refresh(ctx context.Context, raw string) (Session, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback()
	var user platform.Actor
	var session string
	err = tx.QueryRowContext(ctx, "SELECT s.id,u.id,u.email,u.rol FROM sesiones s JOIN usuarios u ON u.id=s.usuario_id WHERE s.refresh_hash=? AND s.expires_at>? FOR UPDATE", platform.Digest(raw), time.Now().Unix()).Scan(&session, &user.ID, &user.Email, &user.Role)
	if err != nil {
		return Session{}, ErrCredentials
	}
	fresh := platform.RandomSecret()
	if _, err = tx.ExecContext(ctx, "UPDATE sesiones SET refresh_hash=? WHERE id=?", platform.Digest(fresh), session); err != nil {
		return Session{}, err
	}
	access, err := s.token(user, session)
	if err != nil {
		return Session{}, err
	}
	if err = tx.Commit(); err != nil {
		return Session{}, err
	}
	return Session{access, user, fresh}, nil
}

/**
 * Revoca la sesión asociada al hash de refresh.
 *
 * @method Logout
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} refresh - Valor de refresh requerido por la operación.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Logout(ctx context.Context, refresh string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM sesiones WHERE refresh_hash=?", platform.Digest(refresh))
	return err
}

/**
 * Lista identidades administrativas sin exponer hashes ni credenciales.
 *
 * @method Users
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {[]platform.Actor, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Users(ctx context.Context) ([]platform.Actor, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,email,rol FROM usuarios ORDER BY email LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []platform.Actor{}
	for rows.Next() {
		var user platform.Actor
		if err = rows.Scan(&user.ID, &user.Email, &user.Role); err != nil {
			return nil, err
		}
		result = append(result, user)
	}
	return result, rows.Err()
}

/**
 * Cambia un rol conservando al menos un administrador mediante locks SQL.
 *
 * @method Role
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} id - Identificador del recurso solicitado.
 * @param {string} role - Valor de role requerido por la operación.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Role(ctx context.Context, id, role string) error {
	if role != "USUARIO" && role != "ADMINISTRADOR" {
		return ErrValidation
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id FROM usuarios WHERE rol='ADMINISTRADOR' FOR UPDATE")
	if err != nil {
		return err
	}
	admins := []string{}
	for rows.Next() {
		var value string
		if err = rows.Scan(&value); err != nil {
			rows.Close()
			return err
		}
		admins = append(admins, value)
	}
	rows.Close()
	if len(admins) == 1 && admins[0] == id && role != "ADMINISTRADOR" {
		return ErrConflict
	}
	res, err := tx.ExecContext(ctx, "UPDATE usuarios SET rol=? WHERE id=?", role, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("usuario inexistente: %w", ErrConflict)
	}
	return tx.Commit()
}
