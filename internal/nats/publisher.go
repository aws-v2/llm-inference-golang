package nats

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	conn   *nats.Conn
	prefix string
}

func NewPublisher(conn *nats.Conn, prefix string) *Publisher {
	return &Publisher{
		conn:   conn,
		prefix: prefix,
	}
}

// Request-response (used for inference)
func (p *Publisher) Request(subject string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// dev.v1.s3.create_presigned_url

	fullSubject := fmt.Sprintf("%s.%s", p.prefix, subject)
	log.Println("Full subject:", fullSubject)
	log.Println("Full payload:", string(data))


	msg, err := p.conn.Request(fullSubject, data, 30*time.Second)
	if err != nil {
		return nil, err
	}

	return msg.Data, nil
}

// Fire-and-forget (for events like "model.registered")
func (p *Publisher) Publish(subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	fullSubject := fmt.Sprintf("%s.%s", p.prefix, subject)

	return p.conn.Publish(fullSubject, data)
}