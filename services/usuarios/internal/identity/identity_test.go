package identity

import (
	"strings"
	"testing"
)

/**
 * Comprueba el comportamiento y los invariantes indicados por ArgonSaltAndVerification.
 *
 * @method TestArgonSaltAndVerification
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestArgonSaltAndVerification(t *testing.T) {
	a, err := Hash("segura-con-12-caracteres")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Hash("segura-con-12-caracteres")
	if a == b || !strings.HasPrefix(a, "$argon2id$") || !Check("segura-con-12-caracteres", a) || Check("incorrecta", a) || Check("x", a+"corrupto") {
		t.Fatal("salt o verificación incorrecta")
	}
	if _, err = Hash("corta"); err == nil {
		t.Fatal("aceptó contraseña corta")
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por EmailCanonicalAndDisplayNamesRejected.
 *
 * @method TestEmailCanonicalAndDisplayNamesRejected
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestEmailCanonicalAndDisplayNamesRejected(t *testing.T) {
	value, err := Normalize(" USUARIO@Ejemplo.com ")
	if err != nil || value != "usuario@ejemplo.com" {
		t.Fatal(value, err)
	}
	for _, email := range []string{"Nombre <usuario@ejemplo.com>", "invalido", "a@b\nX: y"} {
		if _, err := Normalize(email); err == nil {
			t.Fatal("aceptó", email)
		}
	}
}
