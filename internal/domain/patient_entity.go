package domain

import (
	"time"

	"github.com/google/uuid"
)

// Patient represents a patient in the system.
type Patient struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	DateOfBirth time.Time `json:"date_of_birth" db:"date_of_birth"`
	Email       string    `json:"email" db:"email"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
