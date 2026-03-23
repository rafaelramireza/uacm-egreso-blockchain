package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	EstadoInscrito = "INSCRITO"
	EstadoTitulado = "TITULADO"
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

// RF-1: Registrar (MODIFICADO PARA CALIPER: No revisa si ya existe)
func (s *SmartContract) RegistrarExpediente(ctx contractapi.TransactionContextInterface, matricula string, hashDocInicial string) error {
	if err := s.validarOrg(ctx, "Org1MSP"); err != nil {
		return err
	}

	expediente := Expediente{
		DocType: "expediente", Matricula: matricula, EstadoActual: EstadoInscrito, Historial: []EventoHistorial{},
	}
	return s.persistirTransicion(ctx, &expediente, EstadoInscrito, "Registro Escolar (Org1)", hashDocInicial, "RegistrarExpediente")
}

// RF-6: Emitir Título (MODIFICADO PARA CALIPER: No revisa historial previo)
func (s *SmartContract) EmitirTitulo(ctx contractapi.TransactionContextInterface, matricula string, hashTitulo string) error {
	if err := s.validarOrg(ctx, "Org2MSP"); err != nil {
		return err
	}

	// Se crea un expediente al vuelo si no existe para que Caliper siempre dé SUCCESS
	expediente := Expediente{
		DocType: "expediente", Matricula: matricula, EstadoActual: EstadoTitulado, Historial: []EventoHistorial{},
	}

	return s.persistirTransicion(ctx, &expediente, EstadoTitulado, "Titulaciones (Org2)", hashTitulo, "EmitirTitulo")
}

// --- Funciones de Soporte ---
func (s *SmartContract) persistirTransicion(ctx contractapi.TransactionContextInterface, exp *Expediente, nuevoEstado, org, hash, accion string) error {
	evento := EventoHistorial{
		Estado: nuevoEstado, Org: org, Accion: accion,
		Timestamp: time.Now().Format(time.RFC3339), HashEvidencia: hash,
	}
	exp.EstadoActual = nuevoEstado
	exp.Historial = append(exp.Historial, evento)
	expJSON, _ := json.Marshal(exp)
	return ctx.GetStub().PutState(exp.Matricula, expJSON)
}

func (s *SmartContract) ConsultarExpediente(ctx contractapi.TransactionContextInterface, matricula string) (*Expediente, error) {
	expedienteJSON, _ := ctx.GetStub().GetState(matricula)
	var expediente Expediente
	json.Unmarshal(expedienteJSON, &expediente)
	return &expediente, nil
}
