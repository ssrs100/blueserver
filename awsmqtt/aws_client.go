package awsmqtt

import (
	"encoding/json"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/influxdb"
	"path/filepath"
)

var (
	reportChan chan Shadow
	shadowChan chan Shadow
	awsClient  *Client
)

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
				var rd influxdb.ReportData
				if err := json.Unmarshal(s, &rd); err != nil {
					logs.Error("err:%s", err.Error())
					continue
				}
				if err := influxdb.Insert("temperature", &rd); err != nil {
					logs.Error("%s", err.Error())
					continue
				}
				logs.Debug("insert influxdb success")
				if d, err := influxdb.GetLatest("temperature", "DC0D30AABB02"); err != nil {
					logs.Error("%s", err.Error())
					continue
				} else {
					logs.Debug("latest:%v", *d)
				}

			}
		}
	}
}
