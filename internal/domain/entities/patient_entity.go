package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm" // Added GORM import
)

// Patient represents a patient in the system.
type Patient struct {
	ID          uuid.UUID `json:"id" db:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name        string    `json:"name" db:"name" gorm:"not null"`
	DateOfBirth time.Time `json:"date_of_birth" db:"date_of_birth" gorm:"not null"`
	Email       string    `json:"email" db:"email" gorm:"unique;not null"`
	CreatedAt   time.Time `json:"created_at" db:"created_at" gorm:"not null"` // gorm will default to autoCreateTime
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at" gorm:"not null"` // gorm will default to autoUpdateTime
}
