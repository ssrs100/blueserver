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
	"strconv"
	"strings"
	"time"
)

var SupportTypes = make(map[string]string)

func init() {
	SupportTypes["IBEACON"] = "IBEACON"
	SupportTypes["EDDYSTONE_UID"] = "EDDYSTONE_UID"
	SupportTypes["ALTBEACON"] = "ALTBEACON"
	SupportTypes["EDDYSTONE_EID"] = "EDDYSTONE_EID"
}

type Beacon struct {
	Id          string `json:"id"`
	DeviceId    string `json:"device_id"`
	Type        string `json:"type"`
	ProjectId   string `json:"project_id"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

const (
	ACTIVE   = "ACTIVE"
	INACTIVE = "INACTIVE"
)

func (b *Beacon) dbObjectTrans(beacon bluedb.Beacon) Beacon {
	b1 := Beacon{
		Id:          beacon.Id,
		DeviceId:    beacon.DeviceId,
		Type:        beacon.Type,
		ProjectId:   beacon.ProjectId,
		Status:      beacon.Status,
		Description: beacon.Description,
	}
	return b1
}

func (b *Beacon) dbListObjectTrans(beacons []bluedb.Beacon) []Beacon {
	ret := make([]Beacon, 0)
	for _, v := range beacons {
		ret = append(ret, b.dbObjectTrans(v))
	}
	return ret
}

type CreateBeaconResponse struct {
	BeanId string `json:"id"`
}

func RegisterBeacon(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var beanReq = &Beacon{}
	err = json.Unmarshal(body, beanReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	// check beacon info
	devType := getType(beanReq.Type)
	if len(devType) == 0 || len(beanReq.DeviceId) == 0 {
		strErr := fmt.Sprintf("Invalid type(%s) or device_id(%s).", beanReq.Type, beanReq.DeviceId)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}
	projectId := ps.ByName("projectId")
	if len(projectId) == 0 {
		strErr := fmt.Sprintf("Invalid project_id(%s).", projectId)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	if beanReq.Status != ACTIVE && beanReq.Status != INACTIVE {
		strErr := fmt.Sprintf("Invalid status(%s).", beanReq.Status)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	params := make(map[string]interface{})
	params["type"] = devType
	params["project_id"] = projectId
	params["device_id"] = beanReq.DeviceId
	bean := bluedb.QueryBeacons(params)
	if len(bean) > 0 {
		strErr := fmt.Sprintf("Beacon(%s) has been registed.", beanReq.DeviceId)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	logs.Info("register beacon:%v", beanReq)
	beanDb := bluedb.Beacon{
		"",
		beanReq.DeviceId,
		devType,
		projectId,
		beanReq.Status,
		beanReq.Description,
		time.Now(),
	}
	beanId := bluedb.CreateBeacon(beanDb)
	if len(beanId) <= 0 {
		strErr := fmt.Sprintf("Beacon(%s) regiter fail.", beanReq.DeviceId)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateBeaconResponse{
		BeanId: beanId,
	})
	w.WriteHeader(http.StatusOK)
}

func getType(t string) string {
	if devType, ok := SupportTypes[strings.ToUpper(t)]; ok {
		return devType
	}
	return ""
}

func DeleteBeacon(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("beaconId")
	params := make(map[string]interface{})
	params["beacon_id"] = id
	if attachments := bluedb.QueryAttachments(params); len(attachments) > 0 {
		strErr := fmt.Sprintf("Beacon(%s) has some attachments ,that should be deleted first.", id)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}
	err := bluedb.DeleteBeacon(id)
	if err != nil {
		logs.Error("Delete beacon failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

}

func ListBeacons(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	projectId := ps.ByName("projectId")
	params := make(map[string]interface{})
	params["project_id"] = projectId

	if limit, err := strconv.Atoi(req.Form.Get("limit")); err == nil {
		params["limit"] = limit
	}

	if offset, err := strconv.Atoi(req.Form.Get("offset")); err == nil {
		params["offset"] = offset
	}

	beancons := bluedb.QueryBeacons(params)

	logs.Debug("list beancons:%v", beancons)
	var b = Beacon{}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b.dbListObjectTrans(beancons))
}

func UpdateBeacon(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var beanReq = &Beacon{}
	err = json.Unmarshal(body, beanReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	// check
	id := ps.ByName("beaconId")
	if len(id) == 0 {
		logs.Error("update bean fail, id should be set.")
		DefaultHandler.ServeHTTP(w, req, errors.New("Beacon id should be set."), http.StatusBadRequest)
		return
	}

	//TODO: check device

	beancon := bluedb.Beacon{
		Id:          id,
		Description: beanReq.Description,
	}
	err = bluedb.UpdateBeacon(beancon)
	if err != nil {
		logs.Error("Delete beacon failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

}

func ActiveBeancon(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("beaconId")
	if b, _ := bluedb.QueryBeaconById(id); b == nil {
		strErr := fmt.Sprintf("Active beacon failed: %v not exist.", id)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}
	err := bluedb.UpdateBeaconStatus(id, ACTIVE)
	if err != nil {
		logs.Error("Active beacon failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeActiveBeancon(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("beaconId")
	if b, _ := bluedb.QueryBeaconById(id); b == nil {
		strErr := fmt.Sprintf("DeActive beacon failed: %v not exist.", id)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}
	err := bluedb.UpdateBeaconStatus(id, INACTIVE)
	if err != nil {
		logs.Error("DeActive beacon failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
