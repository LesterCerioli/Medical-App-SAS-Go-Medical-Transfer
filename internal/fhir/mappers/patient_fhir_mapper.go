package mappers

import (
	"encoding/json"
	"fmt"

	// No direct import for "time" here, relying on entities.Patient using time.Time

	"medical-record-service/internal/domain/entities"
	// "github.com/google/uuid" // Se IDs FHIR forem UUIDs
)

type FHIRHumanName struct {
	Use    string   `json:"use,omitempty"`    // usual | official | temp | nickname | anonymous | old | maiden
	Family string   `json:"family,omitempty"` // Family name (often surname)
	Given  []string `json:"given,omitempty"`  // Given names (not including surname)

}

type FHIRPatientGender string

const (
	GenderMale    FHIRPatientGender = "male"
	GenderFemale  FHIRPatientGender = "female"
	GenderOther   FHIRPatientGender = "other"
	GenderUnknown FHIRPatientGender = "unknown"
)

type FHIRPatientResource struct {
	ResourceType string            `json:"resourceType"`
	ID           string            `json:"id,omitempty"`
	Name         []FHIRHumanName   `json:"name,omitempty"`
	BirthDate    string            `json:"birthDate,omitempty"`
	Gender       FHIRPatientGender `json:"gender,omitempty"`
}

func MapPatientToFHIR(patient entities.Patient, fhirVersion string) (json.RawMessage, error) {
	if patient.Name == "" {
		return nil, fmt.Errorf("patient name is required for FHIR mapping")
	}

	humanName := FHIRHumanName{
		Use:   "official",
		Given: []string{patient.Name},
	}

	fhirPatient := FHIRPatientResource{
		ResourceType: "Patient",
		ID:           patient.ID.String(),
		Name:         []FHIRHumanName{humanName},
		BirthDate:    patient.DateOfBirth.Format("2006-01-02"),
	}

	rawJSON, err := json.MarshalIndent(fhirPatient, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling FHIR patient resource to JSON: %w", err)
	}

	return rawJSON, nil
}
