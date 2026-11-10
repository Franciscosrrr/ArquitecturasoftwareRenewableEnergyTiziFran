"""Comprobaciones estáticas de contratos publicados y referencias versionadas."""
import hashlib,json,unittest
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]/'docs/contracts'

class ContractAssets(unittest.TestCase):
    """/**
     * Verifica la integridad de contratos y referencias del repositorio.
     *
     * @method ContractAssets
     * @returns {ContractAssets} Casos estáticos previos a las llamadas HTTP.
     */
    """
    def test_shared_v101_is_byte_frozen(self):
        """/**
         * Comprueba que ampliar la web no modifica el contrato compartido con Clínica.
         *
         * @method test_shared_v101_is_byte_frozen
         * @returns {None} Falla ante cualquier cambio del OpenAPI1.0.1 salvo finales de línea.
         */
        """
        raw=(ROOT/'v1/openapi.json').read_bytes().replace(b'\r\n',b'\n')
        self.assertEqual(hashlib.sha256(raw).hexdigest(),'3876121082fa76cd7ca6b25defbc7edc45ea0a4a89b3ee851fcf681e446ffda3')

    def test_all_contract_references_resolve(self):
        """/**
         * Resuelve JSON Pointer y archivos relativos de todos los contratos procesables.
         *
         * @method test_all_contract_references_resolve
         * @returns {None} Falla si una referencia apunta a un esquema o archivo inexistente.
         */
        """
        for path in ROOT.rglob('*.json'):
            doc=json.loads(path.read_text(encoding='utf-8-sig'))
            pending=[doc]
            while pending:
                node=pending.pop()
                if isinstance(node,list):pending.extend(node)
                elif isinstance(node,dict):
                    pending.extend(node.values())
                    if '$ref' not in node:continue
                    address=node['$ref']
                    if '://' in address:continue
                    target,_,pointer=address.partition('#')
                    target_doc=json.loads((path.parent/target).read_text(encoding='utf-8-sig')) if target else doc
                    for part in pointer.strip('/').split('/') if pointer.strip('/') else []:
                        key=part.replace('~1','/').replace('~0','~')
                        target_doc=target_doc[int(key)] if isinstance(target_doc,list) else target_doc[key]

if __name__=='__main__':unittest.main()
