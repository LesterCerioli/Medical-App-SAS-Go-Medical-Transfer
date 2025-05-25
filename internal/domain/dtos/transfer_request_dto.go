package dtos

import "github.com/google/uuid"

type TransferRequest struct {
	SourceFacilityID      string    `json:"source_facility_id" validate:"required"`
	DestinationFacilityID string    `json:"destination_facility_id" validate:"required"`
	PatientID             uuid.UUID `json:"patient_id" validate:"required"`
}
