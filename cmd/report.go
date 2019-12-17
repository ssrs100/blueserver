package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/awsmqtt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"time"
)

type PayLoad struct {
	Device      string  `json:"device"`
	Timestamp   int  `json:"timestamp"`
	Rssi        int  `json:"rssi"`
	Temperature float32 `json:"temperature"`
	Humidity    float32 `json:"humidity"`
	DeviceName  string  `json:"device_name"`
	Power       string  `json:"power"`
}

func main() {
	baseDir := utils.GetBasePath()
	certDir := filepath.Join(baseDir, "conf", "cert")
	u := "admin"
	c := conf.LoadFile(filepath.Join(certDir, u, "conf.json"))
	if c == nil {
		log.Fatal("load report.json fail")
	}
	iotEndpoint := c.GetString("iot_endpoint")
	tlsCert, err := tls.LoadX509KeyPair(filepath.Join(certDir, u, "certificate.pem.crt"), filepath.Join(certDir, u, "private.pem.key"))

	certs := x509.NewCertPool()

	caPem, err := ioutil.ReadFile(filepath.Join(certDir, "AmazonRootCA1.pem"))
	if err != nil {
		log.Fatal(err.Error())
	}

	certs.AppendCertsFromPEM(caPem)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		RootCAs:      certs,
	}
	awsServerURL := fmt.Sprintf("ssl://%s:8883", iotEndpoint)

	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(awsServerURL)
	mqttOpts.SetMaxReconnectInterval(3 * time.Second)
	mqttOpts.SetClientID("report-cli")
	mqttOpts.SetTLSConfig(tlsConfig)

	cli := mqtt.NewClient(mqttOpts)
	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error().Error())
	}
	defer cli.Disconnect(100)

	rpConfig := conf.LoadFile(filepath.Join(baseDir, "bin", "report.json"))
	if rpConfig == nil {
		log.Fatal("no report.json found")
	}
	topic := rpConfig.GetString("topic")
	device := rpConfig.GetString("device")
	rssi := rpConfig.GetInt("rssi")
	temp := rpConfig.GetFloat("temperature")
	humidity := rpConfig.GetFloat("humidity")
	deviceName := rpConfig.GetString("device_name")
	power := rpConfig.GetString("power")

	t := time.Now().Unix()
	timestamp := rpConfig.GetIntWithDefault("timestamp", int(t)) * 1000
	pl := PayLoad{
		Device:      device,
		Timestamp:   timestamp,
		Rssi:        rssi,
		Temperature: float32(temp),
		Humidity:    float32(humidity),
		DeviceName:  deviceName,
		Power:       power,
	}
	data, err := json.Marshal(&pl)
	if err != nil {
		log.Fatal("marshal fail, err:", err.Error())
	}
	
	// listen
	reportChan, err := subscribeForThingReport(topic, cli)
	if err != nil {
		log.Fatal("subscribe thing fail, err:", err.Error())
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go listen(reportChan, &wg)
	res := cli.Publish(topic, 0, false, data)
	if res.WaitTimeout(time.Second*5) && res.Error() != nil {
		log.Fatal("no report.json found", res.Error())
	}
	wg.Wait()
}


func subscribeForThingReport(topic string, cli mqtt.Client) (chan *awsmqtt.Shadow, error) {
	shadowChan := make(chan *awsmqtt.Shadow)
	token := cli.Subscribe(
		topic,
		0,
		func(client mqtt.Client, msg mqtt.Message) {
			tpc := msg.Topic()
			thing := tpc[len("$aws/things/") : len(tpc)-len("/reports")]
			s := awsmqtt.Shadow{
				Msg:   msg.Payload(),
				Thing: thing,
			}
			shadowChan <- &s
		},
	)
	token.Wait()

	return shadowChan, token.Error()
}

func listen(reportChan chan *awsmqtt.Shadow, wg *sync.WaitGroup) {
	t := time.NewTimer(10 * time.Second)
	select {
	case _, ok := <-reportChan:
		if !ok {
			log.Println("failed to read from shadow channel")
		} else {
			log.Println("success to receive thing")
		}
	case <-t.C:
		log.Println("timeout, fail to receive thing")
	}
	wg.Done()
}

