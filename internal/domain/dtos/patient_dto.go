package dtos

import (
	"time"

	"github.com/google/uuid"
)

type PatientDTO struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DateOfBirth string    `json:"date_of_birth"` // Formatted as YYYY-MM-DD
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
