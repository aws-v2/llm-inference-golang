package nats

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"llm-inference-service/internal/sse"

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

// InstanceEvent mirrors the payload the EC2 service publishes for each
// lifecycle transition.
type InstanceEvent struct {
	EventType  string          `json:"event_type"`
	SessionID  string          `json:"session_id"`
	InstanceID string          `json:"instance_id,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

// SubscribeInstanceEvents listens on {prefix}.ec2.events.instance.*
// The EC2 service publishes per-session subjects like:
//
//	{prefix}.ec2.events.instance.{sessionID}
//
// Each incoming message is decoded and forwarded to the SSE hub so
// that the matching browser client receives the event.
func (s *Subscriber) SubscribeInstanceEvents(hub *sse.Hub) error {
	// Wildcard: catches any session-specific subject.
	// dev.v1.ec2.instance.lifecycle
	fullSubject := fmt.Sprintf("%s.ec2.instance.lifecycle", s.prefix)

	_, err := s.conn.Subscribe(fullSubject, func(msg *nats.Msg) {
		// Extract sessionID from the last token of the subject.
		parts := strings.Split(msg.Subject, ".")
		sessionID := parts[len(parts)-1]

		// Attempt to parse so we can log the event type.
		var event InstanceEvent
		if jsonErr := json.Unmarshal(msg.Data, &event); jsonErr != nil {
			log.Printf("[SSE] failed to parse instance event: %v", jsonErr)
		} else {
			log.Printf("[SSE] event=%s session=%s instance=%s",
				event.EventType, event.SessionID, event.InstanceID)
			// Prefer the session ID embedded in the payload when available.
			if event.SessionID != "" {
				sessionID = event.SessionID
			}
		}

		hub.Send(sessionID, msg.Data)
	})

	if err != nil {
		return fmt.Errorf("subscribe instance events: %w", err)
	}

	log.Println("[SSE] Subscribed to", fullSubject)
	return nil
}