package awsmqtt

import (
	"encoding/json"
	"fmt"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/kuzemkon/aws-iot-device-sdk-go/device"
	"path/filepath"
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
	baseDir := utils.GetBasePath()
	client, err := device.NewClient(
		device.KeyPair{
			PrivateKeyPath:    filepath.Join(baseDir, "conf", "private.pem.key"),
			CertificatePath:   filepath.Join(baseDir, "conf", "certificate.pem.crt"),
			CACertificatePath: filepath.Join(baseDir, "conf", "AmazonRootCA1.pem"),
		},
		conf.GetStringWithDefault("iot_endpoint", "a359ikotxsoxw8-ats.iot.us-west-2.amazonaws.com"), // AWS IoT endpoint
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
					logs.Error("err:", err.Error())
					continue
				}
				fmt.Println(string(s))
			}
		}
	}
}
