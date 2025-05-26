package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log" // Usar um logger mais estruturado no futuro
	"sync" // Adicionado para sync.WaitGroup
	"time" // Adicionado para time.Sleep

	"medical-record-service/internal/adapters"
	"medical-record-service/internal/domain/dtos"
	// "medical-record-service/internal/domain/entities" // REMOVED: Para buscar Patient
	"medical-record-service/internal/domain/repositories"
	"medical-record-service/internal/fhir/mappers" // Para o PatientFHIRMapper

	"github.com/google/uuid"
)

const PatientExportQueue = "patient_export_jobs"

// TransferServiceImpl implementa TransferServiceContract.
type TransferServiceImpl struct {
	patientRepo     repositories.PatientRepositoryContract
	queueAdapter    adapters.QueueAdapter
	logger          *log.Logger
	serviceCtx      context.Context    // Contexto para controlar o ciclo de vida dos consumidores
	serviceCancel   context.CancelFunc // Função para cancelar o serviceCtx
	consumersWg     sync.WaitGroup     // Para esperar consumidores no Stop, se necessário
}

// NewTransferService cria uma nova instância de TransferServiceImpl.
func NewTransferService(
	patientRepo repositories.PatientRepositoryContract,
	queueAdapter adapters.QueueAdapter,
	logger *log.Logger,
) TransferServiceContract {
	ctx, cancel := context.WithCancel(context.Background())
	return &TransferServiceImpl{
		patientRepo:   patientRepo,
		queueAdapter:  queueAdapter,
		logger:        logger,
		serviceCtx:    ctx,
		serviceCancel: cancel,
	}
}

// Start inicia os consumidores de longa duração do serviço.
func (s *TransferServiceImpl) Start(ctx context.Context) error {
	s.logger.Println("TransferService starting...")
	
	err := s.queueAdapter.StartConsuming(s.serviceCtx, PatientExportQueue, s.handlePatientExportJob)
	if err != nil {
		s.logger.Printf("Erro ao iniciar consumidor para a fila '%s': %v\n", PatientExportQueue, err)
		return fmt.Errorf("falha ao iniciar consumidor para %s: %w", PatientExportQueue, err)
	}
	s.logger.Printf("Consumidor para a fila '%s' iniciado.\n", PatientExportQueue)
	
	return nil
}

// Stop para os consumidores de longa duração do serviço.
func (s *TransferServiceImpl) Stop(ctx context.Context) error {
	s.logger.Println("TransferService stopping...")
	
	s.serviceCancel() 
	
	s.logger.Println("TransferService stopped.")
	return nil
}


// ExportJobData define a estrutura dos dados para uma tarefa de exportação na fila.
type ExportJobData struct {
	ExportID    string          `json:"exportId"`
	PatientFHIR json.RawMessage `json:"patientFhir"`
	FHIRVersion string          `json:"fhirVersion"`
	IsEmergency bool            `json:"isEmergency"`
	PatientID   string          `json:"patientId"`
}

// InitiateExport busca os dados do paciente, mapeia para FHIR e enfileira para exportação.
func (s *TransferServiceImpl) InitiateExport(ctx context.Context, request dtos.InitiateTransferRequest) (string, error) {
	s.logger.Printf("Iniciando exportação para paciente ID: %s, FHIR Version: %s, Emergência: %t\n",
		request.PatientID, request.FHIRVersion, request.IsEmergency)

	patient, err := s.patientRepo.GetByID(ctx, request.PatientID)
	if err != nil {
		s.logger.Printf("Erro ao buscar paciente %s: %v\n", request.PatientID, err)
		return "", fmt.Errorf("paciente não encontrado %s: %w", request.PatientID, err)
	}
	if patient == nil {
		s.logger.Printf("Paciente %s não encontrado (nil retornado sem erro explícito)\n", request.PatientID)
		return "", fmt.Errorf("paciente %s não encontrado", request.PatientID)
	}

	patientFHIRBytes, err := mappers.MapPatientToFHIR(*patient, request.FHIRVersion)
	if err != nil {
		s.logger.Printf("Erro ao mapear paciente %s para FHIR %s: %v\n", request.PatientID, request.FHIRVersion, err)
		return "", fmt.Errorf("erro no mapeamento FHIR para paciente %s: %w", request.PatientID, err)
	}

	exportID := uuid.New().String()
	jobData := ExportJobData{
		ExportID:    exportID,
		PatientFHIR: patientFHIRBytes,
		FHIRVersion: request.FHIRVersion,
		IsEmergency: request.IsEmergency,
		PatientID:   request.PatientID.String(),
	}

	jobBytes, err := json.Marshal(jobData)
	if err != nil {
		s.logger.Printf("Erro ao serializar jobData para JSON para exportID %s: %v\n", exportID, err)
		return "", fmt.Errorf("erro ao preparar dados para fila: %w", err)
	}

	err = s.queueAdapter.Publish(ctx, PatientExportQueue, jobBytes)
	if err != nil {
		s.logger.Printf("Erro ao publicar job de exportação %s na fila '%s': %v\n", exportID, PatientExportQueue, err)
		return "", fmt.Errorf("erro ao enfileirar job de exportação: %w", err)
	}

	s.logger.Printf("Job de exportação %s para paciente %s enfileirado com sucesso na fila '%s'.\n",
		exportID, request.PatientID, PatientExportQueue)

	return exportID, nil
}

// handlePatientExportJob é o handler para processar jobs da fila de exportação de pacientes.
func (s *TransferServiceImpl) handlePatientExportJob(ctx context.Context, jobData []byte) error {
    s.logger.Printf("Processando job de exportação. Tamanho dos dados: %d bytes\n", len(jobData))

    var jobPayload ExportJobData
    if err := json.Unmarshal(jobData, &jobPayload); err != nil {
        s.logger.Printf("[ERRO] Falha ao deserializar job de exportação: %v. Dados: %s\n", err, string(jobData))
        return fmt.Errorf("falha ao deserializar jobData: %w", err)
    }

    s.logger.Printf("[INFO] Job de Exportação Recebido: ExportID=%s, PatientID=%s, FHIRVersion=%s, IsEmergency=%t\n",
        jobPayload.ExportID, jobPayload.PatientID, jobPayload.FHIRVersion, jobPayload.IsEmergency)

    // Logar o bundle FHIR (pode ser grande)
    // Para evitar spam no log em produção, poderia ser um log de debug ou truncado.
    // Por agora, vamos logar uma parte ou um indicador.
    fhirBundleStr := string(jobPayload.PatientFHIR)
    logMsg := fmt.Sprintf("[INFO] Conteúdo FHIR para ExportID %s (primeiros 200 chars): %.200s", jobPayload.ExportID, fhirBundleStr)
    if len(fhirBundleStr) > 200 {
        logMsg += "..."
    }
    s.logger.Println(logMsg)

    // Simular trabalho de processamento da exportação
    s.logger.Printf("[INFO] Simulando processamento da exportação para ExportID %s...\n", jobPayload.ExportID)
    time.Sleep(1 * time.Second) // Simula o tempo que levaria para "exportar"

    s.logger.Printf("[INFO] Processamento do job de exportação ExportID %s concluído.\n", jobPayload.ExportID)
    return nil
}
