package services

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"errors" 
	"fmt" // Added missing fmt import

	"medical-record-service/internal/adapters" // Para MockQueueAdapter
	"medical-record-service/internal/domain/dtos"
	"medical-record-service/internal/domain/entities" // Still needed for patient in tests
	"medical-record-service/internal/domain/repositories" // For PatientRepositoryContract
	fhirmappers "medical-record-service/internal/fhir/mappers" 

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert" 
	// "github.com/stretchr/testify/mock" // Removed as MockPatientRepository from here is removed
)

// Note: MockPatientRepository definition and its methods have been moved to mocks_test.go

// --- Mock QueueAdapter ---
type MockQueueAdapter struct {
	PublishFunc        func(ctx context.Context, queueName string, jobData []byte) error
	StartConsumingFunc func(ctx context.Context, queueName string, handler adapters.JobHandler) error
	StopConsumingFunc  func(ctx context.Context, queueName string) error
	
	PublishedMessages map[string][][]byte 
	mu                sync.Mutex
	Handlers          map[string]adapters.JobHandler 
}

func NewMockQueueAdapter() *MockQueueAdapter {
	return &MockQueueAdapter{
		PublishedMessages: make(map[string][][]byte),
		Handlers:          make(map[string]adapters.JobHandler),
	}
}

func (m *MockQueueAdapter) Publish(ctx context.Context, queueName string, jobData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, queueName, jobData)
	}
	m.PublishedMessages[queueName] = append(m.PublishedMessages[queueName], jobData)
	log.Printf("MockQueueAdapter: Message published to %s", queueName)
	return nil
}

func (m *MockQueueAdapter) StartConsuming(ctx context.Context, queueName string, handler adapters.JobHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.StartConsumingFunc != nil {
		return m.StartConsumingFunc(ctx, queueName, handler)
	}
	m.Handlers[queueName] = handler 
	log.Printf("MockQueueAdapter: Consumer started for %s", queueName)
	return nil
}

func (m *MockQueueAdapter) StopConsuming(ctx context.Context, queueName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.StopConsumingFunc != nil {
		return m.StopConsumingFunc(ctx, queueName)
	}
	log.Printf("MockQueueAdapter: Consumer stopped for %s", queueName)
	return nil
}


// --- Testes ---
func TestNewTransferService(t *testing.T) {
	logger := log.New(os.Stdout, "TestNewTransferService: ", log.LstdFlags)
	mockPatientRepo := &MockPatientRepository{} // This will now refer to the mock in mocks_test.go
	mockQueueAdapter := NewMockQueueAdapter()
	
	svc := NewTransferService(mockPatientRepo, mockQueueAdapter, logger)
	assert.NotNil(t, svc, "NewTransferService should not return nil")
}

func TestTransferService_InitiateExport_Success(t *testing.T) {
	logger := log.New(os.Stdout, "TestInitiateExport: ", log.LstdFlags)
	mockPatientRepo := &MockPatientRepository{} // This will now refer to the mock in mocks_test.go
	mockQueueAdapter := NewMockQueueAdapter()

	patientID := uuid.New()
	patient := &entities.Patient{ // entities.Patient is still needed here for test setup
		ID:          patientID,
		Name:        "Test Patient",
		DateOfBirth: time.Now().AddDate(-30, 0, 0),
	}

	mockPatientRepo.GetByIDFunc = func(ctx context.Context, id uuid.UUID) (*entities.Patient, error) {
		if id == patientID {
			return patient, nil
		}
		return nil, errors.New("patient not found")
	}

	svc := NewTransferService(mockPatientRepo, mockQueueAdapter, logger)

	req := dtos.InitiateTransferRequest{
		PatientID:   patientID,
		FHIRVersion: "STU3",
		IsEmergency: false,
	}

	exportID, err := svc.InitiateExport(context.Background(), req)
	assert.NoError(t, err, "InitiateExport should not return an error on success")
	assert.NotEmpty(t, exportID, "ExportID should not be empty")

	mockQueueAdapter.mu.Lock()
	publishedJobs, ok := mockQueueAdapter.PublishedMessages[PatientExportQueue]
	mockQueueAdapter.mu.Unlock()

	assert.True(t, ok, "No jobs were published to PatientExportQueue")
	assert.Len(t, publishedJobs, 1, "Expected 1 job to be published")

	var jobData ExportJobData // ExportJobData is defined in transfer_service.go (same package for tests)
	err = json.Unmarshal(publishedJobs[0], &jobData)
	assert.NoError(t, err, "Failed to unmarshal job data from queue")
	assert.Equal(t, exportID, jobData.ExportID)
	assert.Equal(t, patientID.String(), jobData.PatientID)
	assert.Equal(t, "STU3", jobData.FHIRVersion)

	var fhirPatient fhirmappers.FHIRPatientResource 
	err = json.Unmarshal(jobData.PatientFHIR, &fhirPatient)
	assert.NoError(t, err, "Failed to unmarshal PatientFHIR from job data")
	assert.Equal(t, "Patient", fhirPatient.ResourceType)
	assert.Equal(t, patientID.String(), fhirPatient.ID)
}

func TestTransferService_Start_And_HandlePatientExportJob(t *testing.T) {
	logger := log.New(os.Stdout, "TestHandleExportJob: ", log.LstdFlags)
	mockPatientRepo := &MockPatientRepository{} 
	mockQueueAdapter := NewMockQueueAdapter()

	svc := NewTransferService(mockPatientRepo, mockQueueAdapter, logger).(*TransferServiceImpl)

	err := svc.Start(context.Background())
	assert.NoError(t, err, "svc.Start() should not return an error")

	exportID := uuid.New().String()
	patientID := uuid.New()
	// Using fmt.Sprintf, so "fmt" import is needed.
	fhirBundle := json.RawMessage(fmt.Sprintf(`{"resourceType":"Patient","id":"%s","name":[{"given":["Test"]}]}`, patientID.String()))
	
	jobPayload := ExportJobData{
		ExportID:    exportID,
		PatientFHIR: fhirBundle,
		FHIRVersion: "STU3",
		IsEmergency: false,
		PatientID:   patientID.String(),
	}
	jobBytes, _ := json.Marshal(jobPayload)

	var buf strings.Builder
	originalLoggerOutput := logger.Writer()
	logger.SetOutput(&buf)
	defer logger.SetOutput(originalLoggerOutput) 

	mockQueueAdapter.mu.Lock()
	handler, ok := mockQueueAdapter.Handlers[PatientExportQueue]
	mockQueueAdapter.mu.Unlock()
	assert.True(t, ok, "Handler for PatientExportQueue not registered by StartConsuming")

	processingError := handler(context.Background(), jobBytes)
	assert.NoError(t, processingError, "handlePatientExportJob returned an error")
	
	time.Sleep(50 * time.Millisecond)

	logOutput := buf.String()
	assert.Contains(t, logOutput, fmt.Sprintf("Job de Exportação Recebido: ExportID=%s", exportID), "Log should contain export ID")
	assert.Contains(t, logOutput, fmt.Sprintf("Conteúdo FHIR para ExportID %s", exportID), "Log should contain FHIR content indicator")
	assert.Contains(t, logOutput, fmt.Sprintf("Processamento do job de exportação ExportID %s concluído", exportID), "Log should indicate job completion")

	err = svc.Stop(context.Background())
	assert.NoError(t, err, "svc.Stop() should not return an error")
}

func TestTransferService_InitiateExport_PatientNotFound(t *testing.T) {
	logger := log.New(os.Stdout, "TestPatientNotFound: ", log.LstdFlags)
	mockPatientRepo := &MockPatientRepository{} 
	mockQueueAdapter := NewMockQueueAdapter()

	patientID := uuid.New()
	mockPatientRepo.GetByIDFunc = func(ctx context.Context, id uuid.UUID) (*entities.Patient, error) {
		return nil, errors.New("database error: patient not found") 
	}
	
	svc := NewTransferService(mockPatientRepo, mockQueueAdapter, logger)
	req := dtos.InitiateTransferRequest{PatientID: patientID, FHIRVersion: "STU3"}

	_, err := svc.InitiateExport(context.Background(), req)
	assert.Error(t, err, "Expected an error when patient is not found")
	assert.Contains(t, err.Error(), "paciente não encontrado", "Error message should indicate patient not found")
}

func TestTransferService_Stop(t *testing.T) {
    logger := log.New(os.Stdout, "TestServiceStop: ", log.LstdFlags)
    mockPatientRepo := &MockPatientRepository{} 
    mockQueueAdapter := NewMockQueueAdapter()
   
    svc := NewTransferService(mockPatientRepo, mockQueueAdapter, logger).(*TransferServiceImpl)

    err := svc.Start(context.Background())
    assert.NoError(t, err)

    err = svc.Stop(context.Background())
    assert.NoError(t, err)
    
    assert.True(t, true, "svc.Stop executed (further checks on consumer state would be more complex)")
}
