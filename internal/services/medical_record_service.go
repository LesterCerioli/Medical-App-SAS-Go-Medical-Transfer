package services

import (
	"context"
	"log" // Using standard log package for now
	"time"

	"medical-record-service/internal/domain/repositories"
)

// MedicalRecordServiceImpl implements MedicalRecordServiceContract for medical record data processing.
type MedicalRecordServiceImpl struct {
	medicalRecordRepo repositories.MedicalRecordRepositoryContract
	logger            *log.Logger
	stopChan          chan struct{}
}

// NewMedicalRecordService creates a new instance of MedicalRecordServiceImpl.
func NewMedicalRecordService(repo repositories.MedicalRecordRepositoryContract, logger *log.Logger) MedicalRecordServiceContract {
	return &MedicalRecordServiceImpl{
		medicalRecordRepo: repo,
		logger:            logger,
		stopChan:          make(chan struct{}),
	}
}

// Start begins the background processing loop for medical records.
func (s *MedicalRecordServiceImpl) Start(ctx context.Context) error {
	s.logger.Println("Medical Record service started...")

	go func() {
		ticker := time.NewTicker(12 * time.Second) // Example: different interval
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.logger.Println("Medical Record service alive (periodic check)...") // Placeholder
			// Add a call to a repo method to ensure it's "used" for build purposes
			// This can be removed once actual processing logic uses the repo.
			// For example:
			// _, err := s.medicalRecordRepo.ListAll(ctx)
			// if err != nil {
			// 	 s.logger.Printf("Error during ListAll in Medical Record service tick: %v", err)
			// }
			case <-s.stopChan:
				s.logger.Println("Medical Record service stopping background processor...")
				// Example call to use medicalRecordRepo to avoid unused error during build
				// This is a placeholder.
				_, _ = s.medicalRecordRepo.ListAll(context.Background()) // Use new context if ctx might be done
				return
			case <-ctx.Done():
				s.logger.Println("Medical Record service stopping due to context cancellation...")
				return
			}
		}
	}()
	return nil
}

// Stop gracefully shuts down the medical record processing service.
func (s *MedicalRecordServiceImpl) Stop(ctx context.Context) error {
	s.logger.Println("Medical Record service stop requested...")
	close(s.stopChan)
	return nil
}

// ProcessMedicalRecordData handles the processing of new or updated medical record data.
func (s *MedicalRecordServiceImpl) ProcessMedicalRecordData(ctx context.Context, data MedicalRecordData) error {
	s.logger.Printf("Processing medical record data: %+v", data)
	// Example of using the repository.
	// Assumes MedicalRecordData (alias for dtos.CreateMedicalRecordRequest) has PatientID and RecordData.
	// medicalRecordEntity := &entities.MedicalRecord{
	// 	PatientID:  data.PatientID,
	// 	RecordData: data.RecordData, // This is json.RawMessage
	// }
	// err := s.medicalRecordRepo.Create(ctx, medicalRecordEntity)
	// if err != nil {
	// 	s.logger.Printf("Error creating medical record from processed data: %v", err)
	// 	return err
	// }
	// s.logger.Printf("Successfully processed and created medical record: %s for patient %s", medicalRecordEntity.ID, medicalRecordEntity.PatientID)
	
	// Simulate processing
	time.Sleep(100 * time.Millisecond)
	s.logger.Println("Medical record data processing complete for this request.")
	return nil
}
