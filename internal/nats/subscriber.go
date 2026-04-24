package nats

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

type MessageHandler func(data []byte) ([]byte, error)

type Subscriber struct {
	conn   *nats.Conn
	prefix string
}

func NewSubscriber(conn *nats.Conn, prefix string) *Subscriber {
	return &Subscriber{
		conn:   conn,
		prefix: prefix,
	}
}

// Subscribe with request-response handling
func (s *Subscriber) Subscribe(subject string, handler MessageHandler) error {
	fullSubject := fmt.Sprintf("%s.%s", s.prefix, subject)

	_, err := s.conn.Subscribe(fullSubject, func(msg *nats.Msg) {
		resp, err := handler(msg.Data)
		if err != nil {
			log.Println("Handler error:", err)
			return
		}

		if msg.Reply != "" {
			err = s.conn.Publish(msg.Reply, resp)
			if err != nil {
				log.Println("Reply publish error:", err)
			}
		}
	})

	if err != nil {
		return err
	}

	log.Println("Subscribed to", fullSubject)
	return nil
}

// Fire-and-forget subscription (no reply)
func (s *Subscriber) SubscribeAsync(subject string, handler func(data []byte)) error {
	fullSubject := fmt.Sprintf("%s.%s", s.prefix, subject)

	_, err := s.conn.Subscribe(fullSubject, func(msg *nats.Msg) {
		handler(msg.Data)
	})

	if err != nil {
		return err
	}

	log.Println("Subscribed (async) to", fullSubject)
	return nil
}