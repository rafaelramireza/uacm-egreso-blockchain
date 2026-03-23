package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	EstadoInscrito    = "INSCRITO"
	EstadoDocValidado = "DOC_VALIDADO"
	EstadoSSEnCurso   = "SS_EN_CURSO"
	EstadoSSLiberado  = "SS_LIBERADO"
	EstadoCertificado = "CERTIFICADO"
	EstadoTitulado    = "TITULADO"
)

type EventoHistorial struct {
	Estado        string `json:"estado"`
	Org           string `json:"org"`
	Timestamp     string `json:"timestamp"`
	HashEvidencia string `json:"hashEvidencia"`
	Accion        string `json:"accion"`
}

type Expediente struct {
	DocType      string            `json:"docType"`
	Matricula    string            `json:"matricula"`
	EstadoActual string            `json:"estadoActual"`
	Historial    []EventoHistorial `json:"historial"`
}

type SmartContract struct{ contractapi.Contract }

func (s *SmartContract) validarOrg(ctx contractapi.TransactionContextInterface, mspIDEsperado string) error {
	clientMSPID, err := cid.GetMSPID(ctx.GetStub())
	if err != nil {
		return fmt.Errorf("error al obtener identidad: %v", err)
	}
	if clientMSPID != mspIDEsperado {
		return fmt.Errorf("organización no autorizada: requiere %s", mspIDEsperado)
	}
	return nil
}

func (s *SmartContract) RegistrarExpediente(ctx contractapi.TransactionContextInterface, matricula string, hashDocInicial string) error {
	if err := s.validarOrg(ctx, "Org1MSP"); err != nil {
		return err
	}
	existe, _ := s.ExpedienteExiste(ctx, matricula)
	if existe {
		return fmt.Errorf("el expediente %s ya existe", matricula)
	}
	expediente := Expediente{DocType: "expediente", Matricula: matricula, EstadoActual: EstadoInscrito, Historial: []EventoHistorial{}}
	return s.persistirTransicion(ctx, &expediente, EstadoInscrito, "Registro Escolar (Org1)", hashDocInicial, "RegistrarExpediente")
}

func (s *SmartContract) ValidarDocumentos(ctx contractapi.TransactionContextInterface, matricula string, hashCotejo string) error {
	if err := s.validarOrg(ctx, "Org1MSP"); err != nil {
		return err
	}
	exp, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}
	if exp.EstadoActual != EstadoInscrito {
		return fmt.Errorf("transición inválida")
	}
	return s.persistirTransicion(ctx, exp, EstadoDocValidado, "Registro Escolar (Org1)", hashCotejo, "ValidarDocumentos")
}

func (s *SmartContract) IniciarServicioSocial(ctx contractapi.TransactionContextInterface, matricula string, hashAutorizacion string) error {
	if err := s.validarOrg(ctx, "Org2MSP"); err != nil {
		return err
	}
	exp, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}
	if exp.EstadoActual != EstadoDocValidado {
		return fmt.Errorf("transición inválida")
	}
	return s.persistirTransicion(ctx, exp, EstadoSSEnCurso, "Servicio Social (Org2)", hashAutorizacion, "IniciarServicioSocial")
}

func (s *SmartContract) LiberarServicioSocial(ctx contractapi.TransactionContextInterface, matricula string, hashCarta string) error {
	if err := s.validarOrg(ctx, "Org2MSP"); err != nil {
		return err
	}
	exp, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}
	if exp.EstadoActual != EstadoSSEnCurso {
		return fmt.Errorf("transición inválida")
	}
	return s.persistirTransicion(ctx, exp, EstadoSSLiberado, "Servicio Social (Org2)", hashCarta, "LiberarServicioSocial")
}

func (s *SmartContract) EmitirCertificacion(ctx contractapi.TransactionContextInterface, matricula string, hashCert string) error {
	if err := s.validarOrg(ctx, "Org1MSP"); err != nil {
		return err
	}
	exp, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}
	if exp.EstadoActual != EstadoSSLiberado {
		return fmt.Errorf("transición inválida")
	}
	return s.persistirTransicion(ctx, exp, EstadoCertificado, "Certificaciones (Org1)", hashCert, "EmitirCertificacion")
}

func (s *SmartContract) EmitirTitulo(ctx contractapi.TransactionContextInterface, matricula string, hashTitulo string) error {
	if err := s.validarOrg(ctx, "Org2MSP"); err != nil {
		return err
	}
	exp, err := s.ConsultarExpediente(ctx, matricula)
	if err != nil {
		return err
	}
	if exp.EstadoActual != EstadoCertificado {
		return fmt.Errorf("transición inválida")
	}
	if err := s.verificarIntegridad(exp); err != nil {
		return err
	}
	return s.persistirTransicion(ctx, exp, EstadoTitulado, "Titulaciones (Org2)", hashTitulo, "EmitirTitulo")
}

// NUEVA: RF-9 - Búsqueda por Atributo (Rich Query)
func (s *SmartContract) ExpedientesPorEstado(ctx contractapi.TransactionContextInterface, estado string) ([]*Expediente, error) {
	queryString := fmt.Sprintf(`{"selector":{"docType":"expediente","estadoActual":"%s"}}`, estado)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var expedientes []*Expediente
	for resultsIterator.HasNext() {
		res, _ := resultsIterator.Next()
		var exp Expediente
		json.Unmarshal(res.Value, &exp)
		expedientes = append(expedientes, &exp)
	}
	return expedientes, nil
}

func (s *SmartContract) ConsultarExpediente(ctx contractapi.TransactionContextInterface, matricula string) (*Expediente, error) {
	expJSON, _ := ctx.GetStub().GetState(matricula)
	if expJSON == nil {
		return nil, fmt.Errorf("el expediente %s no existe", matricula)
	}
	var exp Expediente
	json.Unmarshal(expJSON, &exp)
	return &exp, nil
}

func (s *SmartContract) persistirTransicion(ctx contractapi.TransactionContextInterface, exp *Expediente, estado, org, hash, accion string) error {
	evento := EventoHistorial{Estado: estado, Org: org, Timestamp: time.Now().UTC().Format(time.RFC3339), HashEvidencia: hash, Accion: accion}
	exp.EstadoActual = estado
	exp.Historial = append(exp.Historial, evento)
	expJSON, _ := json.Marshal(exp)
	return ctx.GetStub().PutState(exp.Matricula, expJSON)
}

func (s *SmartContract) verificarIntegridad(exp *Expediente) error {
	hitos := []string{EstadoInscrito, EstadoDocValidado, EstadoSSEnCurso, EstadoSSLiberado, EstadoCertificado}
	for _, h := range hitos {
		found := false
		for _, e := range exp.Historial {
			if e.Estado == h {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("falta hito: %s", h)
		}
	}
	return nil
}

func (s *SmartContract) ExpedienteExiste(ctx contractapi.TransactionContextInterface, matricula string) (bool, error) {
	expedienteJSON, err := ctx.GetStub().GetState(matricula)
	return err == nil && expedienteJSON != nil, nil
}
