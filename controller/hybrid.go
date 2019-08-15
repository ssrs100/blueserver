package controller

import (
	"encoding/json"
	"github.com/jack0liu/logs"
	"github.com/julienschmidt/httprouter"
	"github.com/ssrs100/blueserver/bluedb"
	"io/ioutil"
	"net/http"
)

type DeviceIdentity struct {
	Type     string `json:"type"`
	DeviceId string `json:"device_id"`
}

type HybridRequest struct {
	Observations    []DeviceIdentity `json:"observations"`
	AttachmentTypes []string         `json:"attachment_types"`
}

type AttachmentResponse struct {
	AttachmentName string `json:"attachment_name"`
	AttachmentType string `json:"attachment_type"`
	Data           string `json:"data"`
}

type Hybrid struct {
	Type        string               `json:"type"`
	DeviceId    string               `json:"device_id"`
	Attachments []AttachmentResponse `json:"attachments"`
}

type HybridResponse struct {
	Beacons []Hybrid `json:"beacons"`
}

func GetForObserved(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var hybridReq = &HybridRequest{}
	err = json.Unmarshal(body, hybridReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	var hybrids = make([]Hybrid, 0)
	for _, ob := range hybridReq.Observations {

		params := make(map[string]interface{})
		params["type"] = getType(ob.Type)
		params["device_id"] = ob.DeviceId
		params["status"] = ACTIVE
		beacons := bluedb.QueryBeacons(params)
		if len(beacons) == 0 {
			continue
		}
		b := beacons[0]
		attachParams := make(map[string]interface{})
		attachParams["beacon_id"] = b.Id
		attachParams["attachment_types"] = hybridReq.AttachmentTypes
		logs.Debug("attachParams:%v", attachParams)
		attachments := bluedb.QueryAttachments(attachParams)
		ams := make([]AttachmentResponse, 0)
		for _, a := range attachments {
			amr := AttachmentResponse{
				AttachmentName: a.AttachmentName,
				AttachmentType: a.AttachmentType,
				Data:           a.Data,
			}
			ams = append(ams, amr)
		}

		h := Hybrid{
			Type:        ob.Type,
			DeviceId:    ob.DeviceId,
			Attachments: ams,
		}

		hybrids = append(hybrids, h)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HybridResponse {
		Beacons: hybrids,
	})
}
