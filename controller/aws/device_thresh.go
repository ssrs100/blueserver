package aws

import (
	"encoding/json"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/common"
	"io/ioutil"
	"net/http"
)

type DevThresh struct {
	ProjectId      string `json:"project_id"`
	Device         string `json:"device"`
	TemperatureMin *int   `json:"temperature_min;omitempty"`
	TemperatureMax *int   `json:"temperature_max;omitempty"`
	HumidityMin    *int   `json:"humidity_min;omitempty"`
	HumidityMax    *int   `json:"humidity_max;omitempty"`
}

func GetDeviceThresh(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	device := ps["device"]
	devt, _ := bluedb.QueryDevThresh(projectId, device)
	if devt == nil {
		logs.Warn("not found thresh data.")
		devt = &bluedb.DeviceThresh{
			DeviceId:       device,
			TemperatureMin: common.MinTemp,
			HumidityMin:    common.MinHumi,
			TemperatureMax: common.MaxTemp,
			HumidityMax:    common.MaxHumi,
		}
	}
	data := DevThresh{
		ProjectId:      projectId,
		Device:         device,
		TemperatureMin: &devt.TemperatureMin,
		TemperatureMax: &devt.TemperatureMax,
		HumidityMin:    &devt.HumidityMin,
		HumidityMax:    &devt.HumidityMax,
	}
	body, err := json.Marshal(data)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(body)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func PutDeviceThresh(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	device := ps["device"]
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer req.Body.Close()

	var devThreshReq = &DevThresh{}
	err = json.Unmarshal(body, devThreshReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	devt, err := bluedb.QueryDevThresh(projectId, device)
	if err != nil {
		logs.Error("get dev thresh fail. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	dt := bluedb.DeviceThresh{}
	if devt == nil {
		if devThreshReq.TemperatureMin != nil {
			dt.TemperatureMin = *devThreshReq.TemperatureMin
		} else {
			dt.TemperatureMin = common.MinTemp
		}
		if devThreshReq.TemperatureMax != nil {
			dt.TemperatureMax = *devThreshReq.TemperatureMax
		} else {
			dt.TemperatureMax = common.MaxTemp
		}
		// humidity
		if devThreshReq.HumidityMin != nil {
			dt.HumidityMin = *devThreshReq.HumidityMin
		} else {
			dt.HumidityMin = common.MinHumi
		}
		if devThreshReq.HumidityMax != nil {
			dt.HumidityMax = *devThreshReq.HumidityMax
		} else {
			dt.HumidityMax = common.MaxHumi
		}
	} else {
		if devThreshReq.TemperatureMin != nil {
			dt.TemperatureMin = *devThreshReq.TemperatureMin
		} else {
			dt.TemperatureMin = devt.TemperatureMin
		}
		if devThreshReq.TemperatureMax != nil {
			dt.TemperatureMax = *devThreshReq.TemperatureMax
		} else {
			dt.TemperatureMax = devt.TemperatureMax
		}
		// humidity
		if devThreshReq.HumidityMin != nil {
			dt.HumidityMin = *devThreshReq.HumidityMin
		} else {
			dt.HumidityMin = devt.HumidityMin
		}
		if devThreshReq.HumidityMax != nil {
			dt.HumidityMax = *devThreshReq.HumidityMax
		} else {
			dt.HumidityMax = devt.HumidityMax
		}
	}
	if err := bluedb.SaveDevThresh(dt); err != nil {
		logs.Error("save dev thresh fail. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
}
