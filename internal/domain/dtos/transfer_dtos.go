package dtos

import (
	"encoding/json"
	"github.com/google/uuid" // Usar UUID para IDs sempre que possível
)

// InitiateTransferRequest é usado para iniciar um novo processo de transferência de paciente.
type InitiateTransferRequest struct {
	PatientID   uuid.UUID `json:"patientId" validate:"required"`
	FHIRVersion string    `json:"fhirVersion" validate:"required,oneof=STU3 DSTU2"` // Exemplo: STU3 ou DSTU2
	IsEmergency bool      `json:"isEmergency"`
	// Adicionar outros campos que possam ser necessários para iniciar, ex:
	// SourceHospitalID string `json:"sourceHospitalId" validate:"required"`
	// DestinationHospitalID string `json:"destinationHospitalId" validate:"required"`
}

// TransferProgress represents common fields for transfer status responses.
type TransferProgress struct {
	TransferID string `json:"transferId"` // ID único para o processo de transferência geral
	Status     string `json:"status"`     // Ex: "PENDING", "IN_PROGRESS", "COMPLETED", "FAILED"
	Message    string `json:"message,omitempty"`
}

// ExportStatusResponse é a resposta para uma operação de exportação.
type ExportStatusResponse struct {
	TransferProgress
	FHIRBundle json.RawMessage `json:"fhirBundle,omitempty"` // O bundle FHIR se a exportação for síncrona e bem-sucedida
	// Ou talvez um link para download se for assíncrono:
	// DownloadURL string `json:"downloadUrl,omitempty"`
}

// ImportStatusResponse é a resposta para uma operação de importação.
type ImportStatusResponse struct {
	TransferProgress
	// Adicionar campos específicos do status da importação, se houver.
	// Ex: RecordsValidated int `json:"recordsValidated"`
	//     RecordsImported int `json:"recordsImported"`
}

// FHIRPatient (exemplo de como poderia ser um recurso FHIR Paciente simplificado)
// Esta struct seria mais complexa e definida em um pacote fhir.mappers
// type FHIRPatient struct {
// 	ResourceType string `json:"resourceType"` // "Patient"
// 	ID           string `json:"id,omitempty"`
// 	Identifier   []struct {
// 		System string `json:"system,omitempty"`
// 		Value  string `json:"value,omitempty"`
// 	} `json:"identifier,omitempty"`
// 	Name []struct {
// 		Family string   `json:"family,omitempty"`
// 		Given  []string `json:"given,omitempty"`
// 	} `json:"name,omitempty"`
// 	// ... outros campos FHIR relevantes
// }
