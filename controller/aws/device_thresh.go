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
	TemperatureMin *int   `json:"temperature_min"`
	TemperatureMax *int   `json:"temperature_max"`
	HumidityMin    *int   `json:"humidity_min"`
	HumidityMax    *int   `json:"humidity_max"`
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
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
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
	logs.Info("body:%s", string(body))
	var devThreshReq DevThresh
	err = json.Unmarshal(body, &devThreshReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	logs.Info("devThreshReq:%v", devThreshReq)
	devt, err := bluedb.QueryDevThresh(projectId, device)
	if err != nil {
		logs.Error("get dev thresh fail. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	dt := bluedb.DeviceThresh{
		ProjectId: projectId,
		DeviceId:  device,
	}
	if devt == nil {
		if devThreshReq.TemperatureMin != nil {
			logs.Info("TemperatureMin:%d", *devThreshReq.TemperatureMin)
			dt.TemperatureMin = *devThreshReq.TemperatureMin
		} else {
			dt.TemperatureMin = common.MinTemp
		}
		if devThreshReq.TemperatureMax != nil {
			logs.Info("TemperatureMax:%d", *devThreshReq.TemperatureMax)
			dt.TemperatureMax = *devThreshReq.TemperatureMax
		} else {
			dt.TemperatureMax = common.MaxTemp
		}
		// humidity
		if devThreshReq.HumidityMin != nil {
			logs.Info("HumidityMin:%d", *devThreshReq.HumidityMin)
			dt.HumidityMin = *devThreshReq.HumidityMin
		} else {
			dt.HumidityMin = common.MinHumi
		}
		if devThreshReq.HumidityMax != nil {
			logs.Info("HumidityMax:%d", *devThreshReq.HumidityMax)
			dt.HumidityMax = *devThreshReq.HumidityMax
		} else {
			dt.HumidityMax = common.MaxHumi
		}
		logs.Info("save to (%v)", dt)
		err = bluedb.SaveDevThresh(dt)
	} else {
		dt.Id = devt.Id
		if devThreshReq.TemperatureMin != nil {
			logs.Info("TemperatureMin:%d", *devThreshReq.TemperatureMin)
			dt.TemperatureMin = *devThreshReq.TemperatureMin
		} else {
			dt.TemperatureMin = devt.TemperatureMin
		}
		if devThreshReq.TemperatureMax != nil {
			logs.Info("TemperatureMax:%d", *devThreshReq.TemperatureMax)
			dt.TemperatureMax = *devThreshReq.TemperatureMax
		} else {
			dt.TemperatureMax = devt.TemperatureMax
		}
		// humidity
		if devThreshReq.HumidityMin != nil {
			logs.Info("HumidityMin:%d", *devThreshReq.HumidityMin)
			dt.HumidityMin = *devThreshReq.HumidityMin
		} else {
			dt.HumidityMin = devt.HumidityMin
		}
		if devThreshReq.HumidityMax != nil {
			logs.Info("HumidityMax:%d", *devThreshReq.HumidityMax)
			dt.HumidityMax = *devThreshReq.HumidityMax
		} else {
			dt.HumidityMax = devt.HumidityMax
		}
		logs.Info("update to (%v)", dt)
		err = bluedb.UpdateDevThresh(dt)
	}
	if err != nil {
		logs.Error("modify dev thresh fail. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
}
