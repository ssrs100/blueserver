package mqttclient

import (
	"bluedb"
	"encoding/json"
	"github.com/satori/go.uuid"
	"model"
	MQTT "pahomqtt"
	"strconv"
	"strings"
	"time"
)

func infoCollect(client MQTT.Client, msg MQTT.Message) {
	topicSegs := strings.Split(msg.Topic(), "/")
	// /GW/00-50-56-C0-00-01/status
	if len(topicSegs) != 4 {
		log.Warn("Topic is not 4, ignore. topic:%s", msg.Topic())
		return
	}
	clientID := topicSegs[2]
	payload := msg.Payload()
	log.Debug("clientID:%s, payload:%v", clientID, payload)
	if len(payload) == 0 {
		log.Error("pay load is 0")
		return
	}

	collections := make([]bluedb.Collection, 0)
	components := strings.Split(string(payload), "\r\n")
	for _, c := range components {
		// +SCAN=1, DC0D30010203,-15,62,00023343405960ADF
		if len(c) <= 6 {
			continue
		}
		ci := c[6:]
		items := strings.Split(ci, ",")
		if len(items) != 5 {
			log.Warn("error format item:%s", c)
			continue
		}
		dbCom := bluedb.QueryComponentByMac(items[1])
		if dbCom == nil {
			log.Warn("device not register")
			continue
		}
		rssi, err := strconv.Atoi(items[2])
		if err != nil {
			log.Error("invalid rssi:%s", items[2])
			continue
		}
		u2, _ := uuid.NewV4()
		component := bluedb.Collection{
			Id:          u2.String(),
			ComponentId: dbCom.Id,
			Rssi:        rssi,
			Data:        items[4],
			CreateAt:    time.Now().UTC(),
		}
		collections = append(collections, component)
	}

	if len(collections) > 0 {
		bluedb.AddBatchCollections(collections)
	}
}

type ActionResponse struct {
	AddrType int
	Addr     string
	Status   int
}

func actionResponse(client MQTT.Client, msg MQTT.Message) {
	topicSegs := strings.Split(msg.Topic(), "/")
	// /GW/00-50-56-C0-00-01/status
	if len(topicSegs) != 4 {
		log.Warn("Topic is not 4, ignore. topic:%s", msg.Topic())
		return
	}
	clientID := topicSegs[2]
	payload := msg.Payload()
	log.Debug("clientID:%s, payload:%v", clientID, payload)
	if len(payload) == 0 {
		log.Error("pay load is 0")
		return
	}

	responses := strings.Split(string(payload), "\r\n")
	for _, c := range responses {
		// +RESPONSE=Addr_type,addr,status\r\n
		if len(c) <= 10 {
			continue
		}
		ci := c[10:]
		items := strings.Split(ci, ",")
		if len(items) != 3 {
			log.Warn("error format item:%s", c)
			continue
		}
		addrType := items[0]
		mac := items[1]
		status := items[2]
		st, err := strconv.Atoi(status)
		if err != nil {
			log.Error("status(%s) is invalid", status)
			continue
		}
		if addrType == "0" {
			addrType = "BEACON"
		} else if addrType == "1" {
			addrType = "GATEWAY"
		} else {
			log.Error("unknown addr type(%s)", addrType)
		}
		dbCom := bluedb.QueryComponentByMacAndType(mac, addrType)
		if dbCom == nil {
			log.Warn("device not register")
			continue
		}

		comDetail, err := bluedb.QueryComponentDetailByComponentId(dbCom.Id)
		if err != nil {
			log.Warn("query detail err:%s", err.Error())
			continue
		}
		// status 1 indicates success
		if st == 1 {
			cd := &model.ComponentDetail{}
			updateData := comDetail.UpdateData
			if err := json.Unmarshal([]byte(updateData), cd); err != nil {
				log.Error("Unmarshal failed, updateData:%s", updateData)
				continue
			}
			comDetail.UpdateStatus = st
			comDetail.UpdateData = ""
			comDetail.TxPower = cd.TxPower
			comDetail.AdvInterval = cd.AdvInterval
			comDetail.ComponentName = cd.ComponentName
			comDetail.Data = cd.Data
			comDetail.Slot = cd.Slot
			bluedb.UpdateComponentDetail(*comDetail)

			com, err := bluedb.QueryComponentById(dbCom.Id)
			if err != nil {
				continue
			}
			com.ComponentPassword = cd.NewPassword
			bluedb.UpdateComponent(*com)

		} else {
			d := bluedb.ComponentDetail{
				Id:           comDetail.Id,
				UpdateStatus: st,
			}
			bluedb.UpdateDetailStatusOnly(d)
		}
	}

}
