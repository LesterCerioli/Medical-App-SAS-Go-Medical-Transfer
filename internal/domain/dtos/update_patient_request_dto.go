package dtos

type UpdatePatientRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	DateOfBirth *string `json:"date_of_birth,omitempty" validate:"omitempty,datetime=2006-01-02"`
	Email       *string `json:"email,omitempty" validate:"omitempty,email"`
}
