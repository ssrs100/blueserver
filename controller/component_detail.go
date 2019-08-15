package controller

import (
	"encoding/json"
	"github.com/jack0liu/logs"
	"github.com/julienschmidt/httprouter"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/common"
	"github.com/ssrs100/blueserver/model"
	"github.com/ssrs100/blueserver/mqttclient"
	"io/ioutil"
	"net/http"
)

type ComponentModifyMsg struct {
	MsgType    string `json:"msg"`
	DstMacType int    `json:"dmac_type"`
	DstMac     string `json:"dmac"`
	Password   string `json:"passwd"`
	Data       string `json:"data"`
}

func UpdateComponentDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var detailReq = &model.ComponentDetail{}
	err = json.Unmarshal(body, detailReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	// check component info
	componentId := ps.ByName("componentId")
	d, err := bluedb.QueryComponentDetailByComponentId(componentId)
	if err != nil {
		logs.Info("component(%s) detail not exist, create it.", componentId)
		createDetail := bluedb.ComponentDetail{
			ComponentId: componentId,
		}
		bluedb.CreateComponentDetail(createDetail)
		if d, err = bluedb.QueryComponentDetailByComponentId(componentId); err != nil {
			logs.Error("update failed. err:%s", err.Error())
			DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
			return
		}
	}

	//if d.UpdateStatus == model.Updating {
	//	strErr := "update failed, current status is updating."
	//	logs.Error(strErr)
	//	DefaultHandler.DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusInternalServerError)
	//	return
	//}

	createDetail := bluedb.ComponentDetail{
		Id:           d.Id,
		ComponentId:  componentId,
		UpdateStatus: model.Updating,
		UpdateData:   detailReq.UpdateData,
	}
	if err = bluedb.UpdateDetailUpdateDataAndStatus(createDetail); err != nil {
		logs.Error("UpdateDetailUpdateDataAndStatus failed. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}

	//notify mqtt
	if mqttclient.Client != nil {
		c, err := bluedb.QueryComponentById(componentId)
		if err != nil {
			logs.Error("QueryComponentById failed. err:%s", err.Error())
		}
		var msg ComponentModifyMsg
		var macType int
		var msgType string
		if c.Type == common.ComponentGatewayType {
			macType = 1
			msgType = "config_gateway_req"
		} else {
			macType = 0
			msgType = "config_beacon_req"
		}
		msg.Data = detailReq.UpdateData
		msg.Password = detailReq.Password
		msg.DstMac = c.MacAddr
		msg.DstMacType = macType
		msg.MsgType = msgType
		body, err := json.Marshal(msg)
		if err != nil {
			logs.Error("marshal err:%s", err.Error())
		}
		logs.Debug("publish body:%s", string(body))
		mqttclient.Client.PublishModify(c.GwMacAddr, body)
	} else {
		logs.Error("mqtt client is nil, not notify")
	}

	w.WriteHeader(http.StatusOK)
}

func GetComponentDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("componentId")

	com, err := bluedb.QueryComponentDetailByComponentId(id)
	if err != nil {
		logs.Error("query component detail failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	var b = model.ComponentDetail{}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b.DbObjectTrans(*com))

}

func CancelUpdateDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("componentId")
	com, err := bluedb.QueryComponentDetailByComponentId(id)
	if err != nil {
		logs.Error("cancel component(%s) detail failed: %v", id, err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	cancelDetail := bluedb.ComponentDetail{
		Id:           com.Id,
		ComponentId:  id,
		UpdateStatus: model.Cancelled,
		UpdateData:   "",
	}

	if err = bluedb.UpdateDetailUpdateDataAndStatus(cancelDetail); err != nil {
		logs.Error("UpdateDetailUpdateDataAndStatus failed. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func SyncComponentDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("componentId")
	_, err := bluedb.QueryComponentDetailByComponentId(id)
	if err != nil {
		logs.Error("sync component(%s) detail failed: %v", id, err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	//TODO

	w.WriteHeader(http.StatusOK)
}
