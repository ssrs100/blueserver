package device

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"

	"time"

	"github.com/eclipse/paho.mqtt.golang"
)

// Thing a structure for working with the AWS IoT device shadows
type Client struct {
	client    mqtt.Client
}

// ThingName the name of the AWS IoT device representation
type ThingName string

// KeyPair the structure contains the path to the AWS MQTT credentials
type KeyPair struct {
	PrivateKeyPath    string
	CertificatePath   string
	CACertificatePath string
}

// Shadow device shadow data
type Shadow []byte

// NewThing returns a new instance of Thing
func NewClient(keyPair KeyPair, awsEndpoint string, clientId string) (*Client, error) {
	tlsCert, err := tls.LoadX509KeyPair(keyPair.CertificatePath, keyPair.PrivateKeyPath)

	certs := x509.NewCertPool()

	caPem, err := ioutil.ReadFile(keyPair.CACertificatePath)
	if err != nil {
		return nil, err
	}

	certs.AppendCertsFromPEM(caPem)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		RootCAs: certs,
	}

	if err != nil {
		return nil, err
	}

	awsServerURL := fmt.Sprintf("ssl://%s:8883", awsEndpoint)

	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(awsServerURL)
	mqttOpts.SetMaxReconnectInterval(1 * time.Second)
	mqttOpts.SetClientID(clientId)
	mqttOpts.SetTLSConfig(tlsConfig)

	c := mqtt.NewClient(mqttOpts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	awsClient := Client {
		client: c,
	}
	return &awsClient, nil
}

// GetThingShadow gets the current thing shadow
func (t *Client) GetThingShadow(thingName string) (Shadow, error) {
	shadowChan := make(chan Shadow)
	errChan := make(chan error)

	if token := t.client.Subscribe(
		fmt.Sprintf("$aws/things/%s/shadow/get/accepted", thingName),
		0,
		func(client mqtt.Client, msg mqtt.Message) {
			shadowChan <- msg.Payload()
		},
	); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	if token := t.client.Subscribe(
		fmt.Sprintf("$aws/things/%s/shadow/get/rejected", thingName),
		0,
		func(client mqtt.Client, msg mqtt.Message) {
			errChan <- errors.New(string(msg.Payload()))
		},
	); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	if token := t.client.Publish(
		fmt.Sprintf("$aws/things/%s/shadow/get", thingName),
		0,
		false,
		[]byte("{}"),
	); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	for {
		select {
		case s, ok := <-shadowChan:
			if !ok {
				return nil, errors.New("failed to read from shadow channel")
			}
			return s, nil
		case err, ok := <-errChan:
			if !ok {
				return nil, errors.New("failed to read from error channel")
			}
			return nil, err
		}
	}
}

// UpdateThingShadow publish a message with new thing shadow
func (t *Client) UpdateThingShadow(thingName string, payload Shadow) error {
	token := t.client.Publish(fmt.Sprintf("$aws/things/%s/shadow/update", thingName), 0, false, []byte(payload))
	token.Wait()
	return token.Error()
}

// SubscribeForThingShadowChanges returns the channel with the shadow updates
func (t *Client) SubscribeForThingShadowChanges() (chan Shadow, error) {
	shadowChan := make(chan Shadow)

	token := t.client.Subscribe(
		fmt.Sprintf("$aws/things/+/shadow/update/accepted"),
		0,
		func(client mqtt.Client, msg mqtt.Message) {
			shadowChan <- msg.Payload()
		},
	)
	token.Wait()

	return shadowChan, token.Error()
}

// SubscribeForThingReport returns the channel with the shadow updates
func (t *Client) SubscribeForThingReport() (chan Shadow, error) {
	shadowChan := make(chan Shadow)

	token := t.client.Subscribe(
		fmt.Sprintf("$aws/things/+/reports"),
		0,
		func(client mqtt.Client, msg mqtt.Message) {
			shadowChan <- msg.Payload()
		},
	)
	token.Wait()

	return shadowChan, token.Error()
}

