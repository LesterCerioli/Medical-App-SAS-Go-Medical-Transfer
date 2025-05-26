package mappers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"medical-record-service/internal/domain/entities"
	"github.com/google/uuid"
)

func TestMapPatientToFHIR_Success(t *testing.T) {
	patientID := uuid.New()
	birthDate := time.Date(1990, time.January, 15, 0, 0, 0, 0, time.UTC)

	patient := entities.Patient{
		ID:          patientID,
		Name:        "John Doe", // Teste com nome simples
		DateOfBirth: birthDate,
		// Adicione um campo Gender à entidade Patient se quiser testar o mapeamento de gênero
		// Gender: "male",
	}

	fhirVersion := "STU3" // Ou DSTU2, o mapeamento atual é genérico

	rawFHIRJson, err := MapPatientToFHIR(patient, fhirVersion)
	if err != nil {
		t.Fatalf("MapPatientToFHIR returned an unexpected error: %v", err)
	}

	if rawFHIRJson == nil {
		t.Fatalf("MapPatientToFHIR returned nil JSON data")
	}

	// Para verificar o conteúdo, podemos desempacotar o JSON para um mapa
	// ou para a nossa struct FHIRPatientResource de teste.
	var fhirPatient FHIRPatientResource
	if err := json.Unmarshal(rawFHIRJson, &fhirPatient); err != nil {
		t.Fatalf("Error unmarshalling rawFHIRJson: %v. JSON: %s", err, string(rawFHIRJson))
	}

	// Verificações básicas
	if fhirPatient.ResourceType != "Patient" {
		t.Errorf("Expected ResourceType 'Patient', got '%s'", fhirPatient.ResourceType)
	}
	if fhirPatient.ID != patientID.String() {
		t.Errorf("Expected FHIR ID '%s', got '%s'", patientID.String(), fhirPatient.ID)
	}
	if fhirPatient.BirthDate != "1990-01-15" {
		t.Errorf("Expected BirthDate '1990-01-15', got '%s'", fhirPatient.BirthDate)
	}

	if len(fhirPatient.Name) != 1 {
		t.Fatalf("Expected 1 name entry, got %d", len(fhirPatient.Name))
	}
	if len(fhirPatient.Name[0].Given) != 1 || fhirPatient.Name[0].Given[0] != "John Doe" {
		t.Errorf("Expected Given name 'John Doe', got '%v'", fhirPatient.Name[0].Given)
	}
	if fhirPatient.Name[0].Use != "official" {
		t.Errorf("Expected Name.Use 'official', got '%s'", fhirPatient.Name[0].Use)
	}

	// Teste de formatação de nome mais complexo (se a lógica de MapPatientToFHIR for atualizada)
	// patientComplexName := entities.Patient{
	// 	ID: uuid.New(), Name: "Doe, John Adam", DateOfBirth: birthDate,
	// }
	// rawFHIRJsonComplex, _ := MapPatientToFHIR(patientComplexName, fhirVersion)
	// ... unmarshal e verificar Family: "Doe", Given: ["John", "Adam"] ...

	// Teste de Gênero (se implementado)
	// if fhirPatient.Gender != GenderMale {
	// 	t.Errorf("Expected Gender 'male', got '%s'", fhirPatient.Gender)
	// }
}

func TestMapPatientToFHIR_NameRequired(t *testing.T) {
	patientID := uuid.New()
	birthDate := time.Date(1990, time.January, 15, 0, 0, 0, 0, time.UTC)

	patient := entities.Patient{
		ID:          patientID,
		Name:        "", // Nome vazio
		DateOfBirth: birthDate,
	}
	fhirVersion := "STU3"

	_, err := MapPatientToFHIR(patient, fhirVersion)
	if err == nil {
		t.Fatalf("MapPatientToFHIR expected an error for missing name, but got nil")
	}
	if !strings.Contains(err.Error(), "patient name is required") {
		t.Errorf("Expected error message to contain 'patient name is required', got: %v", err)
	}
}

// Adicionar mais testes para casos de borda ou diferentes entradas de nome/gênero quando implementados.
