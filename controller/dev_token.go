package controller

import (
	"encoding/json"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"io/ioutil"
	"net/http"
	"time"
)

type DevToken struct {
	Id          string     `json:"id"`
	UserId      string     `json:"user_id"`
	DevId       string     `json:"dev_id"`
	DeviceToken string     `json:"device_token"`
	UpdateAt    *time.Time `json:"update_at"`
}

type RegisterDevTokenReq struct {
	DevId       string `json:"dev_id"`
	DeviceToken string `json:"device_token"`
}

func RegisterDevToken(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	reqBody := RegisterDevTokenReq{}
	err = json.Unmarshal(body, &reqBody)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	if len(reqBody.DevId) == 0 || len(reqBody.DeviceToken) == 0 {
		logs.Error("Invalid body. err:%v", reqBody)
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	devToken := bluedb.DevToken{
		UserId: projectId,
		DevId: reqBody.DevId,
		DeviceToken: reqBody.DeviceToken,
	}
	if err := bluedb.RegisterDevToken(devToken); err != nil {
		logs.Error("Invalid req body:%v, err:%v", reqBody, err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
