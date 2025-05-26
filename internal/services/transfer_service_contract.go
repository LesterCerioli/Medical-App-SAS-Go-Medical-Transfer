package services

import (
	"context"
	"medical-record-service/internal/domain/dtos"
)

// TransferServiceContract define as operações para o serviço de transferência de pacientes.
type TransferServiceContract interface {
	Start(ctx context.Context) error // Para iniciar consumidores de fila, etc.
	Stop(ctx context.Context) error  // Para parar consumidores de fila, etc.

	// InitiateExport inicia o processo de exportação de dados de um paciente.
	// Retorna um ID único para a operação de exportação e um erro, se houver.
	InitiateExport(ctx context.Context, request dtos.InitiateTransferRequest) (exportID string, err error)

	// TODO: Adicionar método para InitiateImport quando essa parte for implementada.
	// InitiateImport(ctx context.Context, request dtos.InitiateImportRequest) (importID string, err error)
}
