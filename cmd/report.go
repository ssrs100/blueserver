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
	"github.com/ssrs100/blueserver/influxdb"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
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

	configJson := rpConfig.GetJson()
	var rpData influxdb.ReportDataList
	if err := json.Unmarshal([]byte(configJson), &rpData); err != nil {
		log.Fatal("unmarshal fail, err:", err.Error())
	}
	for _, d := range rpData.Objects {
		t := time.Now().Unix()
		d.Timestamp = t * 1000
	}
	data, err := json.Marshal(&rpData)
	if err != nil {
		log.Fatal("marshal fail, err:", err.Error())
	}
	
	// listen
	reportChan, err := subscribeForThingReport(topic, cli)
	if err != nil {
		log.Fatal("subscribe thing fail, err:", err.Error())
	}

	echoTopic := strings.Replace(topic, "reports", "echo", -1)
	echoChan, err := subscribeForThingEcho(echoTopic, cli)
	if err != nil {
		log.Fatal("subscribe echo thing fail, err:", err.Error())
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go listen(reportChan, &wg)
	go listen(echoChan, &wg)
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

func subscribeForThingEcho(topic string, cli mqtt.Client) (chan *awsmqtt.Shadow, error) {
	shadowChan := make(chan *awsmqtt.Shadow)
	token := cli.Subscribe(
		topic,
		0,
		func(client mqtt.Client, msg mqtt.Message) {
			tpc := msg.Topic()
			thing := tpc[len("$aws/things/") : len(tpc)-len("/echo")]
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
		os.Exit(1)
	}
	wg.Done()
}

