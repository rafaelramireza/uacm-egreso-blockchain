package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Estados válidos según Sección 4.3 del SRS
const (
	EstadoInscrito    = "INSCRITO"
	EstadoDocValidado = "DOC_VALIDADO"
	EstadoSSEnCurso   = "SS_EN_CURSO"
	EstadoSSLiberado  = "SS_LIBERADO"
	EstadoCertificado = "CERTIFICADO"
	EstadoTitulado    = "TITULADO"
)

// EventoHistorial: Estructura para auditoría (Sección 4.2)
type EventoHistorial struct {
	Estado        string `json:"estado"`
	Org           string `json:"org"`
	Timestamp     string `json:"timestamp"`
	HashEvidencia string `json:"hashEvidencia"`
	Accion        string `json:"accion"`
}

// Expediente: Estructura principal sin PII (RNF-8 y Sección 4.1)
type Expediente struct {
	DocType      string            `json:"docType"`
	Matricula    string            `json:"matricula"`
	EstadoActual string            `json:"estadoActual"`
	Historial    []EventoHistorial `json:"historial"` // Slice ordenado para trazabilidad secuencial
}

type SmartContract struct {
	contractapi.Contract
}

// validarOrg: Validación de identidades institucionales (Sección 5 y RNF-5/6)
func (s *SmartContract) validarOrg(ctx contractapi.TransactionContextInterface, mspIDEsperado string) error {
	clientMSPID, err := cid.GetMSPID(ctx.GetStub())
	if err != nil {
		return fmt.Errorf("error al obtener identidad: %v", err)
	}
	if clientMSPID != mspIDEsperado {
		return fmt.Errorf("organización no autorizada: la función requiere %s", mspIDEsperado)
	}
	return nil
}

// RF-1: Registrar expediente inicial
func (s *SmartContract) RegistrarExpediente(ctx contractapi.TransactionContextInterface, matricula string, hashDocInicial string) error {
	if err := s.validarOrg(ctx, "RegistroMSP"); err != nil {
		return err
	}

	exists, _ := s.ExpedienteExiste(ctx, matricula)
	if exists {
		return fmt.Errorf("el expediente con matrícula %s ya existe", matricula)
	}

	expediente := Expediente{
		DocType:      "expediente",
		Matricula:    matricula,
		EstadoActual: EstadoInscrito,
		Historial:    []EventoHistorial{},
	}

	return s.persistirTransicion(ctx, &expediente, EstadoInscrito, "RegistroMSP", hashDocInicial, "RegistrarExpediente")
}

// RF-2: Validar documentos iniciales
func (s *SmartContract) ValidarDocumentos(ctx contractapi.TransactionContextInterface, matricula string, hashValidacion string) error {
	if err := s.validarOrg(ctx, "RegistroMSP"); err != nil {
		return err
	}

	expediente, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}

	if expediente.EstadoActual != EstadoInscrito {
		return fmt.Errorf("transición inválida: se esperaba %s", EstadoInscrito)
	}

	return s.persistirTransicion(ctx, expediente, EstadoDocValidado, "RegistroMSP", hashValidacion, "ValidarDocumentos")
}

// RF-3: Iniciar Servicio Social
func (s *SmartContract) IniciarServicioSocial(ctx contractapi.TransactionContextInterface, matricula string, hashAutorizacion string) error {
	if err := s.validarOrg(ctx, "ServicioSocialMSP"); err != nil {
		return err
	}

	expediente, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}

	if expediente.EstadoActual != EstadoDocValidado {
		return fmt.Errorf("transición inválida: se esperaba %s", EstadoDocValidado)
	}

	return s.persistirTransicion(ctx, expediente, EstadoSSEnCurso, "ServicioSocialMSP", hashAutorizacion, "IniciarServicioSocial")
}

// RF-4: Liberar Servicio Social
func (s *SmartContract) LiberarServicioSocial(ctx contractapi.TransactionContextInterface, matricula string, hashLiberacion string) error {
	if err := s.validarOrg(ctx, "ServicioSocialMSP"); err != nil {
		return err
	}

	expediente, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}

	if expediente.EstadoActual != EstadoSSEnCurso {
		return fmt.Errorf("transición inválida: se esperaba %s", EstadoSSEnCurso)
	}

	return s.persistirTransicion(ctx, expediente, EstadoSSLiberado, "ServicioSocialMSP", hashLiberacion, "LiberarServicioSocial")
}

// RF-5: Emitir Certificación de Créditos
func (s *SmartContract) EmitirCertificacion(ctx contractapi.TransactionContextInterface, matricula string, hashCertificacion string) error {
	if err := s.validarOrg(ctx, "CertificacionMSP"); err != nil {
		return err
	}

	expediente, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}

	if expediente.EstadoActual != EstadoSSLiberado {
		return fmt.Errorf("transición inválida: se esperaba %s", EstadoSSLiberado)
	}

	return s.persistirTransicion(ctx, expediente, EstadoCertificado, "CertificacionMSP", hashCertificacion, "EmitirCertificacion")
}

// RF-6: Emitir Título Profesional
func (s *SmartContract) EmitirTitulo(ctx contractapi.TransactionContextInterface, matricula string, hashTitulo string) error {
	if err := s.validarOrg(ctx, "TitulacionMSP"); err != nil {
		return err
	}

	expediente, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}

	if expediente.EstadoActual != EstadoCertificado {
		return fmt.Errorf("transición inválida: se esperaba %s", EstadoCertificado)
	}

	// Validación interna de integridad (Requisito crítico RF-6)
	if err := s.verificarIntegridadHitosPrevios(expediente); err != nil {
		return fmt.Errorf("FALLO DE INTEGRIDAD CRIPTOGRÁFICA: %v", err)
	}

	return s.persistirTransicion(ctx, expediente, EstadoTitulado, "TitulacionMSP", hashTitulo, "EmitirTitulo")
}

// RF-7 y RF-8: Consultar expediente e historial
func (s *SmartContract) ConsultarExpediente(ctx contractapi.TransactionContextInterface, matricula string) (*Expediente, error) {
	expedienteJSON, err := ctx.GetStub().GetState(matricula)
	if err != nil {
		return nil, fmt.Errorf("error al leer world state: %v", err)
	}
	if expedienteJSON == nil {
		return nil, fmt.Errorf("el expediente %s no existe", matricula)
	}

	var expediente Expediente
	err = json.Unmarshal(expedienteJSON, &expediente)
	return &expediente, err
}

// RF-9: Listar expedientes por estado (Rich Query CouchDB)
func (s *SmartContract) ExpedientesPorEstado(ctx contractapi.TransactionContextInterface, estado string) ([]*Expediente, error) {
	queryString := fmt.Sprintf(`{"selector":{"docType":"expediente","estadoActual":"%s"}}`, estado)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var expedientes []*Expediente
	for resultsIterator.HasNext() {
		response, _ := resultsIterator.Next()
		var exp Expediente
		json.Unmarshal(response.Value, &exp)
		expedientes = append(expedientes, &exp)
	}
	return expedientes, nil
}

// --- Funciones de Soporte Internas ---

func (s *SmartContract) persistirTransicion(ctx contractapi.TransactionContextInterface, exp *Expediente, nuevoEstado, org, hash, accion string) error {
	evento := EventoHistorial{
		Estado:        nuevoEstado,
		Org:           org,
		Accion:        accion,
		Timestamp:     time.Now().Format(time.RFC3339),
		HashEvidencia: hash,
	}

	exp.EstadoActual = nuevoEstado
	exp.Historial = append(exp.Historial, evento)

	expJSON, _ := json.Marshal(exp)
	return ctx.GetStub().PutState(exp.Matricula, expJSON)
}

func (s *SmartContract) verificarIntegridadHitosPrevios(expediente *Expediente) error {
	hitosRequeridos := []string{EstadoInscrito, EstadoDocValidado, EstadoSSEnCurso, EstadoSSLiberado, EstadoCertificado}

	for _, hito := range hitosRequeridos {
		encontrado := false
		for _, evento := range expediente.Historial {
			if evento.Estado == hito && evento.HashEvidencia != "" {
				encontrado = true
				break
			}
		}
		if !encontrado {
			return fmt.Errorf("falta evidencia obligatoria del hito: %s", hito)
		}
	}
	return nil
}

func (s *SmartContract) ExpedienteExiste(ctx contractapi.TransactionContextInterface, matricula string) (bool, error) {
	expedienteJSON, err := ctx.GetStub().GetState(matricula)
	if err != nil {
		return false, err
	}
	return expedienteJSON != nil, nil
}
