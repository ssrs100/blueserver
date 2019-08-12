package mqttclient

import (
	"fmt"
	"github.com/kuzemkon/aws-iot-device-sdk-go/device"
)

var (
	reportChan chan device.Shadow
	shadowChan chan device.Shadow
    awsClient *device.Client
)

func InitAwsClient() {
	client, err := device.NewClient(
		device.KeyPair{
			PrivateKeyPath:    "E:\\aws\\cert-iot\\aa4ea84834-private.pem.key",
			CertificatePath:   "E:\\aws\\cert-iot\\aa4ea84834-certificate.pem.crt",
			CACertificatePath: "E:\\aws\\cert-iot\\AmazonRootCA1.pem",
		},
		"a359ikotxsoxw8-ats.iot.us-west-1.amazonaws.com", // AWS IoT endpoint
		"blueserver",
	)
	if err != nil {
		panic(err)
	}
	awsClient = client

	reportChan, err = awsClient.SubscribeForThingReport()
	if err != nil {
		panic(err)
	}

	go startAwsClient()
}

func startAwsClient() {
	for {
		select {
		case s, ok := <-reportChan:
			if !ok {
				fmt.Println("failed to read from shadow channel")
			} else {
				fmt.Println(string(s))
			}
		}
	}
}
