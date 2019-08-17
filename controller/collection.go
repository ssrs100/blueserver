package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"net/http"
	"time"
)

type Collection struct {
	Rssi     int       `json:"rssi"`
	Data     string    `json:"data"`
	CreateAt time.Time `json:"create_at"`
}

func (b *Collection) dbObjectTrans(collection bluedb.Collection) Collection {
	b1 := Collection{
		Rssi:     collection.Rssi,
		Data:     collection.Data,
		CreateAt: collection.CreateAt,
	}
	return b1
}

func (b *Collection) dbListObjectTrans(collections []bluedb.Collection) []Collection {
	ret := make([]Collection, 0)
	for _, v := range collections {
		ret = append(ret, b.dbObjectTrans(v))
	}
	return ret
}

func ListCollections(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	logs.Debug("r.RequestURI:%s", req.RequestURI)
	componentId := ps["componentId"]
	params := make(map[string]interface{})
	params["component_id"] = componentId
	var t1, t2 time.Time
	var err error
	if err := req.ParseForm(); err != nil {
		logs.Error("ParseForm err:%s", err.Error())
	}

	for k, v := range req.Form {
		logs.Debug(fmt.Sprintf("%s, %v", k, v))
	}
	if startTime := req.Form.Get("start_time"); len(startTime) > 0 {
		t1, err = time.Parse("2006-01-02T15:04:05.999999999Z", startTime)
		if err != nil {
			strErr := fmt.Sprintf("Invalid start time format. %s", startTime)
			logs.Error(strErr)
			DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
			return
		}
		params["start_time"] = startTime
	}

	if endTime := req.Form.Get("end_time"); len(endTime) > 0 {
		t2, err = time.Parse("2006-01-02T15:04:05.999999999Z", endTime)
		if err != nil {
			strErr := fmt.Sprintf("Invalid end time format. %s", endTime)
			logs.Error(strErr)
			DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
			return
		}
		params["end_time"] = endTime
	}
	logs.Debug("t1:%v, t2:%v", params["start_time"], params["end_time"])
	if t2.Before(t1) {
		strErr := fmt.Sprintf("time should be set, and end time after begin time.")
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	collections := bluedb.QueryCollections(params)

	logs.Debug("list collections:%v", collections)
	var b = Collection{}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b.dbListObjectTrans(collections))
}
