package awsmqtt

import (
	"encoding/json"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/influxdb"
	"path/filepath"
	"time"
)

var (
	reportChan chan Shadow
	shadowChan chan Shadow
	awsClient  *Client
)

type ReportData struct {
	Device      string `json:"device"`
	Timestamp   int64  `json:"timestamp"`
	Rssi        int    `json:"rssi"`
	Temperature int    `json:"temperature"`
	Humidity    int    `json:"humidity"`
}

func InitAwsClient() {
	baseDir := utils.GetBasePath()
	client, err := NewClient(
		KeyPair{
			PrivateKeyPath:    filepath.Join(baseDir, "conf", "private.pem.key"),
			CertificatePath:   filepath.Join(baseDir, "conf", "certificate.pem.crt"),
			CACertificatePath: filepath.Join(baseDir, "conf", "AmazonRootCA1.pem"),
		},
		conf.GetStringWithDefault("iot_endpoint", "a359ikotxsoxw8-ats.iot.us-west-2.amazonaws.com"), // AWS IoT endpoint
		"blueserverclient",
	)
	if err != nil {
		panic(err)
	}
	awsClient = client

	reportChan, err = awsClient.SubscribeForThingReport()
	if err != nil {
		panic(err)
	}

	startAwsClient()
}

func startAwsClient() {
	for {
		select {
		case s, ok := <-reportChan:
			if !ok {
				logs.Debug("failed to read from shadow channel")
			} else {
				var rd ReportData
				if err := json.Unmarshal(s, &rd); err != nil {
					logs.Error("err:%s", err.Error())
					continue
				}
				fields := make(map[string]interface{})
				fields["device"] = rd.Device
				fields["temperature"] = rd.Temperature
				fields["humidity"] = rd.Humidity
				fields["rssi"] = rd.Rssi
				rdTime := time.Unix(0, rd.Timestamp*1000000)
				if err := influxdb.Insert("temperature", fields, &rdTime); err != nil {
					logs.Error("%s", err.Error())
					continue
				}
				if _, err := influxdb.GetLatest("temperature", "DC0D30AABB02"); err != nil {
					logs.Error("%s", err.Error())
					continue
				}

				logs.Debug("insert influxdb success")
			}
		}
	}
}
