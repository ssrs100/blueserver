package controller

import (
	"bluedb"
	"encoding/json"
	"errors"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"model"
	"net/http"
)

const (
	UpdateSuccess = iota
	Updating
)

func UpdateComponentDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var detailReq = &model.ComponentDetail{}
	err = json.Unmarshal(body, detailReq)
	if err != nil {
		log.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	// check param scope
	if detailReq.AdvInterval < 100 || detailReq.AdvInterval > 10000 ||
		detailReq.TxPower < -128 || detailReq.TxPower > 127 ||
		detailReq.Slot < 0 || detailReq.Slot > 9 {
		strErr := "invalid params scope, check adv_interval(100, 10000), tx_power(-128,127), slot(0,9)"
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	// check component info
	componentId := ps.ByName("componentId")
	d, err := bluedb.QueryComponentDetailByComponentId(componentId)
	if err != nil {
		log.Info("component(%s) detail not exist, create it.", componentId)
		createDetail := bluedb.ComponentDetail{
			ComponentId: componentId,
		}
		bluedb.CreateComponentDetail(createDetail)
		if d, err = bluedb.QueryComponentDetailByComponentId(componentId); err != nil {
			log.Error("update failed. err:%s", err.Error())
			DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
			return
		}
	}

	if d.UpdateStatus == Updating {
		strErr := "update failed, current status is updating."
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusInternalServerError)
		return
	}

	createDetail := bluedb.ComponentDetail{
		Id:           d.Id,
		UpdateStatus: Updating,
		UpdateData:   string(body),
	}
	if err = bluedb.UpdateDetailUpdateDataAndStatus(createDetail); err != nil {
		log.Error("UpdateDetailUpdateDataAndStatus failed. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetComponentDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("componentId")

	com, err := bluedb.QueryComponentDetailByComponentId(id)
	if err != nil {
		log.Error("query component detail failed: %v", err.Error())
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
		log.Error("cancel component(%s) detail failed: %v", id, err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	cancelDetail := bluedb.ComponentDetail{
		Id:           com.Id,
		UpdateStatus: UpdateSuccess,
		UpdateData:   "",
	}

	if err = bluedb.UpdateDetailUpdateDataAndStatus(cancelDetail); err != nil {
		log.Error("UpdateDetailUpdateDataAndStatus failed. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func SyncComponentDetail(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("componentId")
	com, err := bluedb.QueryComponentDetailByComponentId(id)
	if err != nil {
		log.Error("sync component(%s) detail failed: %v", id, err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	cancelDetail := bluedb.ComponentDetail{
		Id:           com.Id,
		UpdateStatus: UpdateSuccess,
		UpdateData:   "",
	}

	if err = bluedb.UpdateDetailUpdateDataAndStatus(cancelDetail); err != nil {
		log.Error("UpdateDetailUpdateDataAndStatus failed. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
