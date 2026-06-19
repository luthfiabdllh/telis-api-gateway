package rabbitmq

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	PublishDocumentTask(ctx context.Context, documentID, filePath string) error
	PublishDeleteTask(ctx context.Context, documentID string) error
	Close() error
}

type publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(url string) (Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Declare queue
	_, err = ch.QueueDeclare(
		"ingestion_queue", // name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return nil, err
	}

	return &publisher{conn: conn, ch: ch}, nil
}

func (p *publisher) PublishDocumentTask(ctx context.Context, documentID, filePath string) error {
	payload := map[string]string{
		"action":      "upload",
		"document_id": documentID,
		"file_path":   filePath,
	}
	body, _ := json.Marshal(payload)

	err := p.ch.PublishWithContext(ctx,
		"",                // exchange
		"ingestion_queue", // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}
	log.Printf("Published document upload task: %s", documentID)
	return nil
}

func (p *publisher) PublishDeleteTask(ctx context.Context, documentID string) error {
	payload := map[string]string{
		"action":      "delete",
		"document_id": documentID,
	}
	body, _ := json.Marshal(payload)

	err := p.ch.PublishWithContext(ctx,
		"",                // exchange
		"ingestion_queue", // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		log.Printf("Failed to publish delete message: %v", err)
		return err
	}
	log.Printf("Published document delete task: %s", documentID)
	return nil
}

func (p *publisher) Close() error {
	p.ch.Close()
	return p.conn.Close()
}
