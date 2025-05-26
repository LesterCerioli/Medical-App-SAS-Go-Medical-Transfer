package handlers

import (
	"context" // Added context for the service call
	"log"     // Ou um logger mais estruturado injetado
	"time"    // Para timeouts, se necessário

	"medical-record-service/internal/domain/dtos"
	"medical-record-service/internal/services" // Para TransferServiceContract

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid" // Para validar PatientID se não for feito pelo DTO/validator
)

type TransferHandler struct {
	transferService services.TransferServiceContract
	logger          *log.Logger
}

func NewTransferHandler(ts services.TransferServiceContract, logger *log.Logger) *TransferHandler {
	return &TransferHandler{
		transferService: ts,
		logger:          logger,
	}
}

func (h *TransferHandler) InitiateExport(c *fiber.Ctx) error {
	h.logger.Println("Recebida requisição para InitiateExport")

	var req dtos.InitiateTransferRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Printf("Erro ao parsear corpo da requisição InitiateExport: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Não foi possível parsear a requisição: " + err.Error(),
		})
	}

	if req.PatientID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "patientId é obrigatório"})
	}
	if req.FHIRVersion != "STU3" && req.FHIRVersion != "DSTU2" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "fhirVersion deve ser STU3 ou DSTU2"})
	}

	h.logger.Printf("Dados da requisição InitiateExport validados: %+v\n", req)

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second) // Exemplo de timeout
	defer cancel()

	exportID, err := h.transferService.InitiateExport(ctx, req)
	if err != nil {
		h.logger.Printf("Erro ao chamar TransferService.InitiateExport: %v\n", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao iniciar o processo de exportação: " + err.Error(),
		})
	}

	h.logger.Printf("Exportação iniciada com sucesso. ExportID: %s\n", exportID)
	// Retornar 202 Accepted, pois o processamento é assíncrono (enfileirado).
	return c.Status(fiber.StatusAccepted).JSON(dtos.ExportStatusResponse{
		TransferProgress: dtos.TransferProgress{
			TransferID: exportID,  // Usar o ID da operação de exportação como TransferID aqui
			Status:     "PENDING", // O status inicial é pendente/enfileirado
			Message:    "Processo de exportação iniciado e enfileirado.",
		},
		FHIRBundle: nil, // Nenhum bundle FHIR retornado imediatamente
	})
}

func RegisterTransferRoutes(app *fiber.App, th *TransferHandler) {
	transferGroup := app.Group("/transfer")
	transferGroup.Post("/export", th.InitiateExport)

}
