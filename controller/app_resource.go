package controller

import (
	"encoding/json"
	"fmt"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"net/http"
)

const (
	AdvertisementType = "adv"
)

type AppRes struct {
	Id   string `json:"id"`
	Url  string `json:"beacon_id"`
	Type string `json:"attachment_name"`
}

func GetAdPic(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	res := bluedb.QueryResByType(AdvertisementType)
	if res == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	app := AppRes{
		Id : res.Id,
		Url: fmt.Sprintf("%s/%s/%s", res.Endpoint, res.Uri, res.ResourceName),
		Type: res.Type,
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	body, err := json.Marshal(&app)
	if err != nil {
		logs.Error("%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(body)
}
