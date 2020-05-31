package mqttclient

import (
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/model"
	"strconv"
	"strings"
	"time"
)

const (
	CollectType = "adv_data_ind"
)

type CollectInfoUnit struct {
	DmacType string `json:"dmac_type"`
	Dmac     string `json:"dmac"`
	Rssi     string `json:"rssi"`
	Data     string `json:"data"`
}

type CollectInfo struct {
	Msg  string            `json:"msg"`
	Gmac string            `json:"gmac"`
	Obj  []CollectInfoUnit `json:"obj"`
}

func infoCollect(client mqtt.Client, msg mqtt.Message) {
	topicSegs := strings.Split(msg.Topic(), "/")
	// /GW/00-50-56-C0-00-01/status
	if len(topicSegs) != 4 {
		logs.Warn("Topic is not 4, ignore. topic:%s", msg.Topic())
		return
	}
	clientID := topicSegs[2]
	payload := msg.Payload()
	logs.Debug("info clientID:%s, payload:%v", clientID, string(payload))
	if len(payload) == 0 {
		logs.Error("pay load is 0")
		return
	}

	// get info
	var info CollectInfo
	err := json.Unmarshal(payload, &info)
	if err != nil {
		logs.Error("Invalid payload. err:%s", err.Error())
		return
	}

	if info.Msg != CollectType {
		logs.Error("Invalid msg:%s", info.Msg)
		return
	}

	collections := make([]bluedb.Collection, 0)
	for _, c := range info.Obj {
		addrType := deviceTypeProto2Name(c.DmacType)
		dbCom := bluedb.QueryComponentByMacAndType(c.Dmac, addrType)
		if dbCom == nil {
			logs.Warn("device(%v) not register", c)
			continue
		}
		rssi, err := strconv.Atoi(c.Rssi)
		if err != nil {
			logs.Error("invalid rssi:%d", rssi)
			continue
		}
		u2 := uuid.NewV4()
		component := bluedb.Collection{
			Id:          u2.String(),
			ComponentId: dbCom.Id,
			Rssi:        rssi,
			Data:        c.Data,
			CreateAt:    time.Now().UTC(),
		}
		collections = append(collections, component)
	}

	if len(collections) > 0 {
		bluedb.AddBatchCollections(collections)
	}
}

type ActionResponse struct {
	Msg      string `json:"msg"`
	DmacType string `json:"dmac_type"`
	Dmac     string `json:"dmac"`
	Result   string `json:"result"`
	//0，	成功
	//1，	鉴权失败
	//2，	没有发现设备
	//3，	密码错误
	//4，	参数错误
	//5，	超时
	//6，	配置异常
}

func actionModifyResponse(client mqtt.Client, msg mqtt.Message) {
	logs.Info("")
	topicSegs := strings.Split(msg.Topic(), "/")
	// /GW/00-50-56-C0-00-01/status/response
	if len(topicSegs) < 4 {
		logs.Warn("Topic is not 4, ignore. topic:%s", msg.Topic())
		return
	}
	clientID := topicSegs[2]
	payload := msg.Payload()
	logs.Debug("clientID:%s, payload:%v", clientID, payload)
	if len(payload) == 0 {
		logs.Error("pay load is 0")
		return
	}

	// get response
	var resp ActionResponse
	err := json.Unmarshal(payload, &resp)
	if err != nil {
		logs.Error("Invalid payload. err:%s", err.Error())
		return
	}

	addrType := deviceTypeProto2Name(resp.DmacType)

	dbCom := bluedb.QueryComponentByMacAndType(resp.Dmac, addrType)
	if dbCom == nil {
		logs.Warn("device(mac:%s, addrType:%s) not register", resp.Dmac, addrType)
		return
	}

	comDetail, err := bluedb.QueryComponentDetailByComponentId(dbCom.Id)
	if err != nil {
		logs.Warn("query detail err:%s", err.Error())
		return
	}
	// status 1 indicates success
	if resp.Result == "0" {
		comDetail.UpdateStatus = model.UpdateSuccess
		comDetail.Data = comDetail.UpdateData
		comDetail.UpdateData = ""
		err = bluedb.UpdateComponentDetail(*comDetail)
		if err != nil {
			logs.Warn("update component detail err:%s", err.Error())
			return
		}
	} else {
		logs.Warn("get result:%s", resp.Result)
		st, err := strconv.Atoi(resp.Result)
		if err != nil {
			logs.Error("err:%s", err.Error())
			return
		}
		d := bluedb.ComponentDetail{
			Id:           comDetail.Id,
			UpdateStatus: st,
		}

		err = bluedb.UpdateDetailStatusOnly(d)
		if err != nil {
			logs.Error("err:%s", err.Error())
			return
		}
	}

}

func deviceTypeProto2Name(proto string) string {
	// get addr type
	var addrType string
	if proto == "0" {
		addrType = "BEACON"
	} else if proto == "1" {
		addrType = "GATEWAY"
	} else {
		logs.Error("unknown addr type(%s)", addrType)
		addrType = ""
	}
	return addrType
}
