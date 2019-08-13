package mqttclient

import (
	"encoding/json"
	"fmt"
	"github.com/kuzemkon/aws-iot-device-sdk-go/device"
)

var (
	reportChan chan device.Shadow
	shadowChan chan device.Shadow
	awsClient  *device.Client
)

type ReportData struct {
	Device      string `json:"device"`
	Timestamp   string `json:"timestamp"`
	Rssi        string `json:"rssi"`
	Temperature string `json:"temperature"`
	Humidity    string `json:"humidity"`
}

func InitAwsClient() {
	client, err := device.NewClient(
		device.KeyPair{
			PrivateKeyPath:    "E:\\aws\\cert-iot\\bc12174c01-private.pem.key",
			CertificatePath:   "E:\\aws\\cert-iot\\bc12174c01-certificate.pem.crt",
			CACertificatePath: "E:\\aws\\cert-iot\\AmazonRootCA1.pem",
		},
		"a359ikotxsoxw8-ats.iot.us-west-2.amazonaws.com", // AWS IoT endpoint
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
				var rd ReportData
				if err := json.Unmarshal(s, &rd); err != nil {
					log.Error("err:", err.Error())
					continue
				}
				fmt.Println(string(s))
			}
		}
	}
}
