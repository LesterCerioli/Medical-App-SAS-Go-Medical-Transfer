package domain

import "github.com/google/uuid"

// TransferRequest defines the payload for the POST /transfer endpoint.
type TransferRequest struct {
	SourceFacilityID      string    `json:"source_facility_id" validate:"required"`
	DestinationFacilityID string    `json:"destination_facility_id" validate:"required"`
	PatientID             uuid.UUID `json:"patient_id" validate:"required"`
	// Additional fields like TransferReason, ConsentDetails could be added here.
}
