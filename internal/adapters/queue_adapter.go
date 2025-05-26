package adapters

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// JobHandler é uma função que processa os dados recebidos da fila.
type JobHandler func(ctx context.Context, data []byte) error

// QueueAdapter define a interface para interações com um sistema de filas.
type QueueAdapter interface {
	// Publish envia uma mensagem (jobData) para a fila especificada.
	Publish(ctx context.Context, queueName string, jobData []byte) error
	// Consume começa a consumir mensagens da fila especificada,
	// chamando o handler para cada mensagem recebida.
	// Esta função deve bloquear ou rodar em background, dependendo da implementação.
	StartConsuming(ctx context.Context, queueName string, handler JobHandler) error
	// StopConsuming para o consumo de mensagens.
	StopConsuming(ctx context.Context, queueName string) error
}

// InMemoryQueueAdapter é uma implementação em memória da QueueAdapter usando canais Go.
type InMemoryQueueAdapter struct {
	queues      map[string]chan []byte
	mu          sync.RWMutex
	logger      *log.Logger
	stopChan    map[string]chan struct{} // Para parar consumidores específicos
	wg          sync.WaitGroup           // Para esperar os consumidores pararem
	consumerCtx context.Context          // Contexto para os consumidores
	cancelFunc  context.CancelFunc       // Para cancelar o contexto dos consumidores
}

// NewInMemoryQueueAdapter cria uma nova InMemoryQueueAdapter.
func NewInMemoryQueueAdapter(logger *log.Logger) QueueAdapter {
	consumerCtx, cancelFunc := context.WithCancel(context.Background())
	return &InMemoryQueueAdapter{
		queues:      make(map[string]chan []byte),
		logger:      logger,
		stopChan:    make(map[string]chan struct{}),
		consumerCtx: consumerCtx,
		cancelFunc:  cancelFunc,
	}
}

func (q *InMemoryQueueAdapter) getOrCreateQueue(queueName string) chan []byte {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.queues[queueName]; !ok {
		q.queues[queueName] = make(chan []byte, 100) // Fila com buffer
		q.stopChan[queueName] = make(chan struct{})
		q.logger.Printf("Fila em memória '%s' criada.\n", queueName)
	}
	return q.queues[queueName]
}

// Publish envia uma mensagem para a fila em memória.
func (q *InMemoryQueueAdapter) Publish(ctx context.Context, queueName string, jobData []byte) error {
	queue := q.getOrCreateQueue(queueName)
	q.logger.Printf("Publicando mensagem na fila '%s' (tamanho atual: %d)\n", queueName, len(queue))
	select {
	case queue <- jobData:
		q.logger.Printf("Mensagem publicada na fila '%s'.\n", queueName)
		return nil
	case <-ctx.Done():
		q.logger.Printf("Contexto cancelado ao tentar publicar na fila '%s': %v\n", queueName, ctx.Err())
		return ctx.Err()
	case <-time.After(2 * time.Second): // Timeout para publicação
		q.logger.Printf("Timeout ao publicar na fila '%s'. Fila possivelmente cheia.\n", queueName)
		return errors.New("timeout publishing to queue: " + queueName)
	}
}

// StartConsuming começa a consumir mensagens de uma fila em memória.
func (q *InMemoryQueueAdapter) StartConsuming(ctx context.Context, queueName string, handler JobHandler) error {
	queue := q.getOrCreateQueue(queueName)
	
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		q.logger.Printf("Iniciando consumidor para a fila '%s'...\n", queueName)
		for {
			select {
			case data, ok := <-queue:
				if !ok {
					q.logger.Printf("Canal da fila '%s' fechado. Consumidor encerrando.\n", queueName)
					return
				}
				q.logger.Printf("Mensagem recebida da fila '%s'. Processando...\n", queueName)
				if err := handler(q.consumerCtx, data); err != nil { // Usar q.consumerCtx
					q.logger.Printf("Erro ao processar mensagem da fila '%s': %v\n", queueName, err)
					// Adicionar lógica de retry ou dead-letter aqui se necessário
				}
			case <-q.stopChan[queueName]:
				q.logger.Printf("Sinal de parada recebido para o consumidor da fila '%s'. Encerrando.\n", queueName)
				return
			case <-q.consumerCtx.Done(): // Parar se o contexto principal do adapter for cancelado
				q.logger.Printf("Contexto do InMemoryQueueAdapter cancelado. Consumidor da fila '%s' encerrando.\n", queueName)
				return
			}
		}
	}()
	return nil
}

// StopConsuming para todos os consumidores e limpa os recursos.
// Para parar um consumidor específico, seria necessário um mecanismo mais granular.
// Esta implementação para todos os consumidores associados a este adapter.
func (q *InMemoryQueueAdapter) StopConsuming(ctx context.Context, queueName string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.logger.Printf("Parando consumidor da fila '%s'...\n", queueName)
	if stopChan, ok := q.stopChan[queueName]; ok {
		close(stopChan) // Sinaliza a goroutine do consumidor para parar
		delete(q.stopChan, queueName) 
	}
	
	// Não fechamos o canal da fila (q.queues[queueName]) aqui,
	// pois pode haver publicadores ainda ativos. O consumidor para de ler.
	// Se quiséssemos parar tudo e limpar:
	// q.cancelFunc() // Cancela o contexto de todos os consumidores
	// q.wg.Wait()    // Espera todos os consumidores terminarem
	// q.logger.Println("Todos os consumidores da InMemoryQueueAdapter pararam.")
	return nil
}

// GlobalStop (ou um método similar) seria necessário para parar todos os consumidores e esperar por eles
// antes da aplicação desligar, chamando cancelFunc e wg.Wait().
// Por enquanto, StopConsuming é por nome de fila.
