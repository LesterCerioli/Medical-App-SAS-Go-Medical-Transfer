package mappers

import (
	"encoding/json"
	"fmt"
	"time" // Para formatar datas se necessário

	"medical-record-service/internal/domain/entities"
	// "github.com/google/uuid" // Se IDs FHIR forem UUIDs
)

// FHIRHumanName represents a FHIR HumanName data type.
type FHIRHumanName struct {
	Use    string   `json:"use,omitempty"`    // usual | official | temp | nickname | anonymous | old | maiden
	Family string   `json:"family,omitempty"` // Family name (often surname)
	Given  []string `json:"given,omitempty"`  // Given names (not including surname)
	// Prefix []string `json:"prefix,omitempty"` // Parts that come before the name
	// Suffix []string `json:"suffix,omitempty"` // Parts that come after the name
	// Period *FHIRPeriod `json:"period,omitempty"` // Time period when name was/is in use
}

// FHIRPatientGender represents the administrative gender of a patient.
// FHIR values: male | female | other | unknown
type FHIRPatientGender string

const (
	GenderMale    FHIRPatientGender = "male"
	GenderFemale  FHIRPatientGender = "female"
	GenderOther   FHIRPatientGender = "other"
	GenderUnknown FHIRPatientGender = "unknown"
)

// FHIRPatientResource represents a simplified FHIR Patient resource.
// Focusing on DSTU2/STU3 compatibility for core demographics.
type FHIRPatientResource struct {
	ResourceType string `json:"resourceType"` // Should be "Patient"
	ID           string `json:"id,omitempty"` // Logical id of this artifact
	// Identifier   []FHIRIdentifier `json:"identifier,omitempty"` // Business identifier
	Name          []FHIRHumanName   `json:"name,omitempty"`
	BirthDate     string            `json:"birthDate,omitempty"` // YYYY-MM-DD
	Gender        FHIRPatientGender `json:"gender,omitempty"`    // male | female | other | unknown
	// Active       *bool             `json:"active,omitempty"`   // Whether this patient's record is in active use
	// Address      []FHIRAddress     `json:"address,omitempty"`
	// Telecom      []FHIRContactPoint `json:"telecom,omitempty"` // Contact details
	// maritalStatus, multipleBirth, deceasedBoolean, deceasedDateTime etc.
}

// MapPatientToFHIR converts an internal Patient entity to a FHIR Patient resource (json.RawMessage).
// fhirVersion can be "DSTU2" or "STU3" - for now, the mapping might be generic enough.
func MapPatientToFHIR(patient entities.Patient, fhirVersion string) (json.RawMessage, error) {
	if patient.Name == "" {
		return nil, fmt.Errorf("patient name is required for FHIR mapping")
	}

	// Basic name mapping (assuming patient.Name is full name)
	// A more sophisticated approach would parse the full name into Family and Given.
	// For now, let's assume patient.Name can be used as a given name,
	// or we might need to adjust how names are stored/parsed.
	// If patient.Name is "FirstName LastName", we could try to split it.
	// As a simple placeholder:
	humanName := FHIRHumanName{
		Use:   "official",
		Given: []string{patient.Name}, // Placeholder: needs better parsing if Name is "Family, Given" or "Given Family"
		// Family: "Smith" // Example if we can derive it
	}

	fhirPatient := FHIRPatientResource{
		ResourceType: "Patient",
		ID:           patient.ID.String(), // Use patient's UUID as FHIR logical ID
		Name:         []FHIRHumanName{humanName},
		BirthDate:    patient.DateOfBirth.Format("2006-01-02"), // FHIR standard date format
		// Gender:    Determine gender if available in entities.Patient, e.g., map to FHIRPatientGender
	}
	
	// Example for gender if you add a Gender field to your entities.Patient
	// switch patient.Gender {
	// case "male":
	// 	fhirPatient.Gender = GenderMale
	// case "female":
	// 	fhirPatient.Gender = GenderFemale
	// default:
	// 	fhirPatient.Gender = GenderUnknown
	// }


	rawJSON, err := json.MarshalIndent(fhirPatient, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling FHIR patient resource to JSON: %w", err)
	}

	return rawJSON, nil
}

// TODO: Add mappers for Allergies, Medications, Procedures.
// These would involve defining similar FHIR resource structs (simplified)
// and mapping functions. Example: FHIRAllergyIntolerance, FHIRMedicationStatement, FHIRProcedure.
