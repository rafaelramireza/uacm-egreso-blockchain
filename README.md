Sistema de Gestión de Egreso UACM (Blockchain)
Este proyecto implementa un Smart Contract (Chaincode) desarrollado en Go para la plataforma Hyperledger Fabric v2.5. Su objetivo es automatizar y asegurar la trazabilidad del proceso de egreso de los alumnos de la Universidad Autónoma de la Ciudad de México (UACM), garantizando la inmutabilidad de los documentos académicos y la segregación de funciones administrativas.

🏗️ Arquitectura del Sistema
El sistema se basa en una Máquina de Estados Finita (FSM) que regula el flujo del expediente digital del alumno. Ningún estado puede ser alcanzado sin haber cumplido los requisitos del nivel anterior, asegurando la integridad del proceso institucional.

Flujo de Estados:
INSCRITO: Registro inicial del alumno en la red.

DOCUMENTOS_VALIDADOS: Verificación de documentos de ingreso (Solo Registro Escolar).

SERVICIO_SOCIAL_LIBERADO: Validación de la prestación del servicio social (Solo oficina de Servicio Social).

ESTUDIOS_CERTIFICADOS: (En desarrollo) Certificación académica final.

TITULADO: (En desarrollo) Emisión del acta de titulación inmutable.

🛡️ Características de Seguridad
Segregación de Funciones (SoD): Uso de la librería cid (Client Identity) para validar el MSPID del emisor. Las acciones de Registro Escolar están bloqueadas para los nodos de otras áreas y viceversa.

Trazabilidad Inmutable: Cada cambio de estado almacena un Hash SHA-256 del documento físico original, el Timestamp exacto de la transacción y la identidad digital del emisor.

Consultas Avanzadas (Rich Queries): Integración con CouchDB para permitir búsquedas complejas por estado, matrícula o nombre, facilitando la auditoría administrativa.

🛠️ Requisitos previos
Docker & Docker Compose

Go 1.21.0 o superior

Hyperledger Fabric Samples (test-network)

WSL 2 (si se trabaja en Windows)

🚀 Instalación y Despliegue
Para desplegar este contrato en una red de prueba de Fabric, sigue estos pasos desde tu terminal:

Bash
# Entrar a la red de prueba
cd fabric-samples/test-network

# Desplegar el contrato
./network.sh deployCC -ccn uacm-contract -ccp ../../uacm-egreso/ -ccl go -c canal-uacm -ccv 1.1
🔍 Ejemplos de Uso
Consultar un expediente por matrícula:

Bash
peer chaincode query -C canal-uacm -n uacm-contract -c '{"Args":["ConsultarExpediente", "2023-001"]}'
Búsqueda avanzada por estado (CouchDB):

Bash
peer chaincode query -C canal-uacm -n uacm-contract -c '{"Args":["QueryExpedientes", "{\"selector\":{\"estadoActual\":\"SERVICIO_SOCIAL_LIBERADO\"}}"]}'