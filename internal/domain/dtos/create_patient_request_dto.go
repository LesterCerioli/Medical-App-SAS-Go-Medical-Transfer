package dtos

// CreatePatientRequest defines the payload for creating a new patient.
type CreatePatientRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	DateOfBirth string `json:"date_of_birth" validate:"required,datetime=2006-01-02"` // ISO 8601 date YYYY-MM-DD
	Email       string `json:"email" validate:"required,email"`
}
