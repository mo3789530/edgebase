package mqtt

import (
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client mqtt.Client
}

func Init(broker string, enabled bool) (*Client, error) {
	if !enabled {
		return nil, nil
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID("control-plane")
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	log.Println("Connected to MQTT broker")
	return &Client{client: client}, nil
}

func (c *Client) Publish(topic string, payload interface{}) error {
	if c == nil || c.client == nil {
		return nil
	}
	// TODO: serialize payload
	token := c.client.Publish(topic, 0, false, payload)
	token.Wait()
	return token.Error()
}
