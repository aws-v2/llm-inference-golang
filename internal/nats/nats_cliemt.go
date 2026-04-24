package nats

import (
	"llm-inference-service/pkg/models"
	"time"

	"github.com/nats-io/nats.go"
)

type Client struct {
	Conn       *nats.Conn
	Publisher  *Publisher
	Subscriber *Subscriber
}

func (c *Client) Close() {
	panic("unimplemented")
}

func (c *Client) Request(subject string, req models.InferenceRequest) ([]byte, error) {
	return c.Publisher.Request(subject, req)
}

func NewClient(url, user, password, prefix string) (*Client, error) {
	nc, err := nats.Connect(
		url,
		nats.UserInfo(user, password),
		nats.Timeout(10*time.Second),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		Conn:       nc,
		Publisher:  NewPublisher(nc, prefix),
		Subscriber: NewSubscriber(nc, prefix),
	}, nil
}
