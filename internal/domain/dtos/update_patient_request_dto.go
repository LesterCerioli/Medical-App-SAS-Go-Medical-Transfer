package dtos

// UpdatePatientRequest defines the payload for updating an existing patient.
// Typically, you might want to use pointers for optional fields or have separate DTOs
// for partial updates (PATCH) vs full updates (PUT).
// For simplicity, this example assumes all fields are optional for update,
// but validation rules might differ.
type UpdatePatientRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	DateOfBirth *string `json:"date_of_birth,omitempty" validate:"omitempty,datetime=2006-01-02"`
	Email       *string `json:"email,omitempty" validate:"omitempty,email"`
}
