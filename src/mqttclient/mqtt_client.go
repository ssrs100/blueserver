package mqttclient

import (
	"bluedb"
	"bytes"
	"conf"
	"encoding/json"
	"fmt"
	"logs"
	"net"
	"net/http"
	MQTT "pahomqtt"
	"pahomqtt/packets"
	"time"
	"utils"
)

var (
	log    = logs.GetLogger()
	Client *MQTTClient
)

const (
	HttpPort = "8080"
	ClientId = "blueserver"
)

// This NoOpStore type implements the go-mqtt/Store interface, which
// allows it to be used by the go-mqtt client library. However, it is
// highly recommended that you do not use this NoOpStore in production,
// because it will NOT provide any sort of guaruntee of message delivery.
type NoOpStore struct {
	// Contain nothing
}

func (store *NoOpStore) Open() {
	// Do nothing
}

func (store *NoOpStore) Put(string, packets.ControlPacket) {
	// Do nothing
}

func (store *NoOpStore) Get(string) packets.ControlPacket {
	// Do nothing
	return nil
}

func (store *NoOpStore) Del(string) {
	// Do nothing
}

func (store *NoOpStore) All() []string {
	return nil
}

func (store *NoOpStore) Close() {
	// Do Nothing
}

func (store *NoOpStore) Reset() {
	// Do Nothing
}

type MQTTClient struct {
	c      MQTT.Client
	config *conf.Config
}

func InitClient(conf *conf.Config) *MQTTClient {
	myNoOpStore := &NoOpStore{}
	//tlsConf := &tls.Config{
	//	InsecureSkipVerify: true,
	//}
	opts := MQTT.NewClientOptions()
	//"tcp://52.8.63.206:1883"
	brokerHost := conf.GetString("mqtt_broker")
	broker := "tcp://" + brokerHost + ":1883"

	opts.AddBroker(broker)
	opts.SetClientID(ClientId)
	opts.SetStore(myNoOpStore)
	admin := bluedb.QueryUserByName("admin")
	if admin == nil {
		log.Error("admin user not found")
		return nil
	}
	opts.SetUsername(admin.Name)
	opts.SetPassword(admin.Passwd)

	Client = &MQTTClient{
		config: conf,
		c:      MQTT.NewClient(opts),
	}
	return Client
}

func (mc *MQTTClient) Start() error {
	if token := mc.c.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	mc.subscribe_init()
	return nil
}

func (mc *MQTTClient) Stop() {
	mc.c.Disconnect(250)
}

func (mc *MQTTClient) subscribe_init() {
	ps := make(map[string]interface{})
	ps["type"] = utils.ComponentGatewayType
	components := bluedb.QueryComponents(ps)
	for _, component := range components {
		topicStatus := fmt.Sprintf("/GW/%s/status", component.MacAddr)
		mc.c.Subscribe(topicStatus, 0, infoCollect)
		topicResponse := fmt.Sprintf("/GW/%s/action/response", component.MacAddr)
		mc.c.Subscribe(topicResponse, 0, actionResponse)
	}
}

func (mc *MQTTClient) Subscribe(clientID string) {
	log.Info("Subscribe clientID:%s", clientID)
	topicStatus := fmt.Sprintf("/GW/%s/status", clientID)
	mc.c.Subscribe(topicStatus, 0, infoCollect)
	topicResponse := fmt.Sprintf("/GW/%s/action/response", clientID)
	mc.c.Subscribe(topicResponse, 0, actionResponse)
}

func (mc *MQTTClient) Publish(clientID string, load interface{}) {
	topic := fmt.Sprintf("/GW/%s/status", clientID)
	token := mc.c.Publish(topic, 0, false, load)
	token.Wait()
}

func (mc *MQTTClient) UnSubscribe(clientID string) {
	log.Info("UnSubscribe clientID:%s", clientID)
	topicStatus := fmt.Sprintf("/GW/%s/status", clientID)
	mc.c.Unsubscribe(topicStatus)
	topicResponse := fmt.Sprintf("/GW/%s/action/response", clientID)
	mc.c.Unsubscribe(topicResponse)
}

func (mc *MQTTClient) NotifyUserAdd(name, password string) {
	brokerHost := mc.config.GetString("mqtt_broker")
	endpoint := "http://" + brokerHost + ":" + HttpPort + "/v1/users"
	log.Debug("Notify user add endpoint:%s", endpoint)
	userMap := make(map[string]interface{})
	userMap["name"] = name
	userMap["password"] = password
	bytesData, err := json.Marshal(userMap)
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Debug("Notify user add bytesData:%s", bytesData)

	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	reader := bytes.NewReader(bytesData)
	request, err := http.NewRequest("POST", endpoint, reader)
	if err != nil {
		log.Error(err.Error())
		return
	}
	//request.Close = true
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := &http.Client{
		Timeout:   time.Second * 30,
		Transport: netTransport,
	}
	resp, err := client.Do(request)
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Info("user add success, user:%s", name)
	} else {
		log.Error("user add failed, user:%s ", name)
	}
}

func (mc *MQTTClient) NotifyUserDelete(name string) {
	brokerHost := mc.config.GetString("mqtt_broker")
	endpoint := "http://" + brokerHost + ":" + HttpPort + "/v1/users/" + name
	log.Debug("Notify user delete endpoint:%s", endpoint)

	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	request, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		log.Error(err.Error())
		return
	}
	client := &http.Client{
		Timeout:   time.Second * 30,
		Transport: netTransport,
	}
	resp, err := client.Do(request)
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Info("user delete success, user:%s", name)
	} else {
		log.Error("user delete failed, user:%s ", name)
	}
}
