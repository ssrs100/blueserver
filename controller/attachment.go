package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jack0liu/logs"
	"github.com/julienschmidt/httprouter"
	"github.com/ssrs100/blueserver/bluedb"
	"io/ioutil"
	"net/http"
	"time"
)

type Attachment struct {
	Id             string `json:"id"`
	BeaconId       string `json:"beacon_id"`
	AttachmentName string `json:"attachment_name"`
	AttachmentType string `json:"attachment_type"`
	Data           string `json:"data"`
}

func (a *Attachment) dbObjectTrans(attachment bluedb.Attachment) Attachment {
	b1 := Attachment{
		Id:             attachment.Id,
		BeaconId:       attachment.BeaconId,
		AttachmentName: attachment.AttachmentName,
		AttachmentType: attachment.AttachmentType,
		Data:           attachment.Data,
	}
	return b1
}

func (a *Attachment) dbListObjectTrans(attachments []bluedb.Attachment) []Attachment {
	ret := make([]Attachment, 0)
	for _, v := range attachments {
		ret = append(ret, a.dbObjectTrans(v))
	}
	return ret
}

type CreateAttachmentResponse struct {
	AttachmentId string `json:"id"`
}

func CreateAttachment(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var attachmentReq = &Attachment{}
	err = json.Unmarshal(body, attachmentReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	// check beacon info

	beaconId := ps.ByName("beaconId")
	bean, _ := bluedb.QueryBeaconById(beaconId)
	if bean == nil {
		strErr := fmt.Sprintf("Beacon(%s) not exist.", beaconId)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	if len(attachmentReq.AttachmentName) == 0 ||
		len(attachmentReq.AttachmentType) == 0 ||
		len(attachmentReq.Data) == 0 {
		strErr := fmt.Sprintf("Invalid attachment, name/type/data should be fully set.")
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	attachmentDb := bluedb.Attachment{
		BeaconId:       beaconId,
		AttachmentName: attachmentReq.AttachmentName,
		AttachmentType: attachmentReq.AttachmentType,
		Data:           attachmentReq.Data,
		Create_at:      time.Now(),
	}
	attachmentId := bluedb.CreateAttachment(attachmentDb)
	if len(attachmentId) <= 0 {
		strErr := fmt.Sprintf("Attachment(%s) regiter fail.", attachmentReq.AttachmentName)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateAttachmentResponse{
		AttachmentId: attachmentId,
	})
	w.WriteHeader(http.StatusOK)
}

func DeleteAttachment(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("attachmentId")
	err := bluedb.DeleteAttachment(id)
	if err != nil {
		logs.Error("Delete attachment failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

}

func DeleteAttachmentByBeacon(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("beaconId")
	err := bluedb.DeleteAttachmentByBeacon(id)
	if err != nil {
		logs.Error("Delete attachment failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func GetAttachmentByBeacon(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("beaconId")
	param := make(map[string]interface{})
	param["beacon_id"] = id
	attachments := bluedb.QueryAttachments(param)
	logs.Debug("list beancons:%v", attachments)
	var a = Attachment{}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a.dbListObjectTrans(attachments))
}
