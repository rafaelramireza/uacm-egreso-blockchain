package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	EstadoInscrito         = "INSCRITO"
	EstadoDocsValidados    = "DOCUMENTOS_VALIDADOS"
	EstadoServicioLiberado = "SERVICIO_SOCIAL_LIBERADO"
	EstadoCertificado      = "CERTIFICADO"
	EstadoTitulado         = "TITULADO"
)

type HashEvidencia struct {
	Hash      string `json:"hash"`
	Timestamp string `json:"timestamp"`
	Emisor    string `json:"emisor"`
}

type Expediente struct {
	DocType      string                   `json:"docType"`
	ID           string                   `json:"id"`
	Nombre       string                   `json:"nombre"`
	EstadoActual string                   `json:"estadoActual"`
	Evidencias   map[string]HashEvidencia `json:"evidencias"`
}

type SmartContract struct {
	contractapi.Contract
}

func (s *SmartContract) validarOrg(ctx contractapi.TransactionContextInterface, mspIDEsperado string) error {
	clientMSPID, err := cid.GetMSPID(ctx.GetStub())
	if err != nil {
		return fmt.Errorf("error al obtener MSPID: %v", err)
	}
	if clientMSPID != mspIDEsperado {
		return fmt.Errorf("autorización denegada: la organización %s no tiene permiso para esta acción", clientMSPID)
	}
	return nil
}

func (s *SmartContract) RegistrarIngreso(ctx contractapi.TransactionContextInterface, id string, nombre string) error {
	err := s.validarOrg(ctx, "Org1MSP")
	if err != nil {
		return err
	}

	existe, err := s.ExpedienteExiste(ctx, id)
	if err != nil {
		return err
	}
	if existe {
		return fmt.Errorf("el expediente del alumno %s ya existe", id)
	}

	expediente := Expediente{
		DocType:      "expediente",
		ID:           id,
		Nombre:       nombre,
		EstadoActual: EstadoInscrito,
		Evidencias:   make(map[string]HashEvidencia),
	}

	expedienteJSON, err := json.Marshal(expediente)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, expedienteJSON)
}

func (s *SmartContract) ValidarDocumentacion(ctx contractapi.TransactionContextInterface, id string, hashDocumentos string) error {
	if err := s.validarOrg(ctx, "Org1MSP"); err != nil {
		return err
	}

	expedienteJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("fallo al leer del world state: %v", err)
	}
	if expedienteJSON == nil {
		return fmt.Errorf("el expediente %s no existe", id)
	}

	var expediente Expediente
	err = json.Unmarshal(expedienteJSON, &expediente)
	if err != nil {
		return err
	}

	if expediente.EstadoActual != EstadoInscrito {
		return fmt.Errorf("transición inválida: se requiere estado %s, actual es %s", EstadoInscrito, expediente.EstadoActual)
	}

	expediente.EstadoActual = EstadoDocsValidados
	evidencia := HashEvidencia{
		Hash:      hashDocumentos,
		Timestamp: time.Now().Format(time.RFC3339),
		Emisor:    "Registro Escolar - Org1",
	}
	expediente.Evidencias[EstadoDocsValidados] = evidencia

	expedienteActualizadoJSON, err := json.Marshal(expediente)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, expedienteActualizadoJSON)
}

func (s *SmartContract) LiberarServicioSocial(ctx contractapi.TransactionContextInterface, id string, hashLiberacion string) error {
	if err := s.validarOrg(ctx, "Org2MSP"); err != nil {
		return err
	}

	expedienteJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("fallo al leer del world state: %v", err)
	}
	if expedienteJSON == nil {
		return fmt.Errorf("el expediente %s no existe", id)
	}

	var expediente Expediente
	err = json.Unmarshal(expedienteJSON, &expediente)
	if err != nil {
		return err
	}

	if expediente.EstadoActual != EstadoDocsValidados {
		return fmt.Errorf("transición inválida: se requiere estado %s, actual es %s", EstadoDocsValidados, expediente.EstadoActual)
	}

	expediente.EstadoActual = EstadoServicioLiberado
	evidencia := HashEvidencia{
		Hash:      hashLiberacion,
		Timestamp: time.Now().Format(time.RFC3339),
		Emisor:    "Servicio Social - Org2",
	}
	expediente.Evidencias[EstadoServicioLiberado] = evidencia

	expedienteActualizadoJSON, err := json.Marshal(expediente)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, expedienteActualizadoJSON)
}

func (s *SmartContract) ConsultarExpediente(ctx contractapi.TransactionContextInterface, id string) (*Expediente, error) {
	expedienteJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("fallo al leer del world state: %v", err)
	}
	if expedienteJSON == nil {
		return nil, fmt.Errorf("el expediente %s no existe", id)
	}

	var expediente Expediente
	err = json.Unmarshal(expedienteJSON, &expediente)
	if err != nil {
		return nil, err
	}

	return &expediente, nil
}

func (s *SmartContract) ExpedienteExiste(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	expedienteJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("error al leer del world state: %v", err)
	}
	return expedienteJSON != nil, nil
}

// QueryExpedientes permite realizar búsquedas complejas usando sintaxis de CouchDB (Selector Queries)
func (s *SmartContract) QueryExpedientes(ctx contractapi.TransactionContextInterface, query string) ([]*Expediente, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return nil, fmt.Errorf("error al ejecutar la consulta: %v", err)
	}
	defer resultsIterator.Close()

	var expedientes []*Expediente
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var expediente Expediente
		err = json.Unmarshal(queryResponse.Value, &expediente)
		if err != nil {
			return nil, err
		}
		expedientes = append(expedientes, &expediente)
	}

	return expedientes, nil
}

// CertificarEstudio valida que el alumno cumplió con los créditos académicos.
// Solo permitido para Registro Escolar (Org1).
func (s *SmartContract) CertificarEstudio(ctx contractapi.TransactionContextInterface, matricula string, hashCertificado string) error {
	// 1. Validar Identidad (Seguridad)
	mspid, _ := ctx.GetClientIdentity().GetMSPID()
	if mspid != "Org1MSP" {
		return fmt.Errorf("autorización denegada: la organización %s no tiene permiso para certificar estudios", mspid)
	}

	// 2. Obtener el expediente actual
	expediente, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}

	// 3. Validar Máquina de Estados (Pre-condición)
	if expediente.EstadoActual != "SERVICIO_SOCIAL_LIBERADO" {
		return fmt.Errorf("error de flujo: el alumno %s debe liberar servicio social antes de certificar estudios", matricula)
	}

	// 4. Actualizar Estado y Evidencias
	expediente.EstadoActual = "ESTUDIOS_CERTIFICADOS"
	expediente.Evidencias["ESTUDIOS_CERTIFICADOS"] = Evidencia{
		Hash:      hashCertificado,
		Timestamp: time.Now().Format(time.RFC3339),
		Emisor:    "Certificaciones - Org1",
	}

	// 5. Guardar en el Ledger
	expedienteJSON, _ := json.Marshal(expediente)
	return ctx.GetStub().PutState(matricula, expedienteJSON)
}

// TitularAlumno registra el acta de examen profesional y cierra el ciclo de egreso.
// Solo permitido para Coordinación de Titulación (Org2).
func (s *SmartContract) TitularAlumno(ctx contractapi.TransactionContextInterface, matricula string, hashActa string) error {
	// 1. Validar Identidad (Seguridad)
	mspid, _ := ctx.GetClientIdentity().GetMSPID()
	if mspid != "Org2MSP" {
		return fmt.Errorf("autorización denegada: la organización %s no tiene permiso para emitir títulos", mspid)
	}

	// 2. Obtener el expediente actual
	expediente, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}

	// 3. Validar Máquina de Estados (Pre-condición)
	if expediente.EstadoActual != "ESTUDIOS_CERTIFICADOS" {
		return fmt.Errorf("error de flujo: el alumno %s no cuenta con estudios certificados para proceder a titulación", matricula)
	}

	// 4. Actualizar Estado Final
	expediente.EstadoActual = "TITULADO"
	expediente.Evidencias["TITULADO"] = Evidencia{
		Hash:      hashActa,
		Timestamp: time.Now().Format(time.RFC3339),
		Emisor:    "Titulaciones - Org2",
	}

	// 5. Guardar en el Ledger
	expedienteJSON, _ := json.Marshal(expediente)
	return ctx.GetStub().PutState(matricula, expedienteJSON)
}
