"""Verifica la selección de restos sintéticos sin acceder a bases ni crear cuentas reales."""
import copy
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('cleanup_tool', Path(__file__).resolve().parents[1] / 'scripts/limpiar-datos-prueba.py')
cleanup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cleanup)


def fixture():
    """/**
     * Construye un inventario aislado con cuentas legítimas, restos y referencias en uso.
     *
     * @method fixture
     * @returns {dict} Datos en memoria que nunca se escriben al entorno de la aplicación.
     */
    """
    return {'mysql-usuarios': {'usuarios': [{'id': 'admin', 'email': 'admin@renewable.local', 'rol': 'ADMINISTRADOR'}, {'id': 'real', 'email': '2407889@ucc.edu.ar', 'rol': 'USUARIO'}, {'id': 'test', 'email': 'prueba-abc@example.test', 'rol': 'USUARIO'}, {'id': 'other', 'email': 'otra@empresa.example', 'rol': 'USUARIO'}], 'sesiones': [{'id': 's-real', 'usuario_id': 'real'}, {'id': 's-test', 'usuario_id': 'test'}]}, 'mysql-consumo': {'lugares': [], 'configuraciones': [{'owner_id': 'real', 'entrada': {'catalogoId': 'verif1234567890abcdef'}}], 'evaluaciones': [], 'idempotencia': [], 'outbox': []}, 'mysql': {'electrodomesticos': [{'id': 'iluminacion', 'categoria': 'Iluminación'}, {'id': 'verif1234567890abcdef', 'categoria': 'Prueba'}, {'id': 'verifabcdef1234567890', 'categoria': 'Prueba'}], 'catalogo_outbox': []}, 'mongo': {'paneles': [{'_id': 'PANEL-SIM-450', 'fuente': 'Referencia didáctica'}, {'_id': 'solar1234567890abcdef', 'fuente': 'Ficha de prueba de búsqueda'}], 'estudios': []}}


class CleanupSelection(unittest.TestCase):
    """/**
     * Protege las cuentas reales, las fichas utilizadas y los datos fuera del plan de borrado.
     *
     * @method CleanupSelection
     * @returns {CleanupSelection} Casos de seguridad del mantenimiento explícito.
     */
    """
    def test_default_only_removes_generated_accounts_and_unused_fixtures(self):
        """/**
         * Limita la limpieza ordinaria a identidades sintéticas y conserva una ficha referenciada.
         *
         * @method test_default_only_removes_generated_accounts_and_unused_fixtures
         * @returns {None} Falla si se seleccionan cuentas legítimas o una ficha todavía utilizada.
         */
        """
        data = fixture()
        selected = cleanup.plan(data)
        self.assertEqual(selected['usuarios'], ['test'])
        self.assertEqual(selected['equipos'], ['verifabcdef1234567890'])
        self.assertEqual(selected['paneles'], ['solar1234567890abcdef'])

    def test_final_only_is_explicit_and_requires_both_protected_accounts(self):
        """/**
         * Exige las dos cuentas autorizadas antes de preparar una entrega con solo esas identidades.
         *
         * @method test_final_only_is_explicit_and_requires_both_protected_accounts
         * @returns {None} Falla si el modo final elimina una cuenta protegida o admite su ausencia.
         */
        """
        data = fixture()
        self.assertEqual(cleanup.plan(data, True)['usuarios'], ['other', 'test'])
        data['mysql-usuarios']['usuarios'] = [u for u in data['mysql-usuarios']['usuarios'] if u['id'] != 'real']
        with self.assertRaises(ValueError):
            cleanup.plan(data, True)

    def test_fingerprint_detects_changes_to_preserved_data(self):
        """/**
         * Detecta una modificación de datos legítimos aunque se hayan quitado correctamente los restos.
         *
         * @method test_fingerprint_detects_changes_to_preserved_data
         * @returns {None} Falla si un cambio en una cuenta o configuración real pasa inadvertido.
         */
        """
        data = fixture()
        selected = cleanup.plan(data)
        expected = cleanup.fingerprint(data, selected)
        changed = copy.deepcopy(data)
        changed['mysql-usuarios']['usuarios'] = [u for u in changed['mysql-usuarios']['usuarios'] if u['id'] != 'test']
        changed['mysql-usuarios']['sesiones'] = [s for s in changed['mysql-usuarios']['sesiones'] if s['usuario_id'] != 'test']
        self.assertEqual(cleanup.fingerprint(changed, selected), expected)
        changed['mysql-consumo']['configuraciones'][0]['entrada']['catalogoId'] = 'otro'
        self.assertNotEqual(cleanup.fingerprint(changed, selected), expected)

    def test_sql_literals_do_not_interpolate_user_text(self):
        """/**
         * Codifica identificadores como hexadecimales en vez de introducir texto en el shell o SQL.
         *
         * @method test_sql_literals_do_not_interpolate_user_text
         * @returns {None} Falla si una comilla o una orden recibida se interpola en el literal.
         */
        """
        literal = cleanup.literals(["a'; DROP TABLE usuarios;--"])
        self.assertNotIn('DROP', literal)
        self.assertNotIn("'", literal)
        self.assertEqual(cleanup.literals([]), 'NULL')


if __name__ == '__main__':
    unittest.main()
