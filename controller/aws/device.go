package aws

import (
	"encoding/json"
	"fmt"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/influxdb"
	"net/http"
	"strings"
	"time"
)


type InnerThresh struct {
	TemperatureMin *float32 `json:"temperature_min"`
	TemperatureMax *float32 `json:"temperature_max"`
	HumidityMin    *float32 `json:"humidity_min"`
	HumidityMax    *float32 `json:"humidity_max"`
}


type DeviceData struct {
	ProjectId   string      `json:"project_id"`
	Thing       string      `json:"thing"`
	Device      string      `json:"device"`
	Timestamp   string      `json:"timestamp"`
	Rssi        json.Number `json:"rssi"`
	Temperature json.Number `json:"temperature"`
	Humidity    json.Number `json:"humidity"`
	DeviceName  string      `json:"device_name"`
	Power       string      `json:"power"`
	Thresh      InnerThresh   `json:"thresh"`
}

func GetDeviceLatestData(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	device := ps["device"]
	data, err := influxdb.GetLatest("temperature", "", device, projectId)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("thing id not found"))
		return
	}
	body, err := json.Marshal(data)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(body)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func GetMultiDeviceLatestData(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	deviceAddrs := req.URL.Query().Get("deviceAddrs")
	if len(deviceAddrs) == 0 {
		logs.Error("deviceAddrs is empty")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("deviceAddrs is empty"))
		return
	}
	logs.Debug("device addr:%s", deviceAddrs)
	datas := make([]*DeviceData, 0)
	deviceList := strings.Split(deviceAddrs, ";")
	for _, device := range deviceList {
		data, err := influxdb.GetLatest("temperature", "", device, projectId)
		if err != nil {
			logs.Error("get device(%s) data. err:%s", device, err.Error())
			continue
		}
		thresh := getDevThresh(projectId, device)
		dd := &DeviceData{
			ProjectId:   data.ProjectId,
			Thing:       data.Thing,
			Device:      data.Device,
			Timestamp:   data.Timestamp,
			Rssi:        data.Rssi,
			Temperature: data.Temperature,
			Humidity:    data.Humidity,
			DeviceName:  data.DeviceName,
			Power:       data.Power,
			Thresh:      InnerThresh{
				TemperatureMin : thresh.TemperatureMin,
				TemperatureMax : thresh.TemperatureMax,
				HumidityMin    : thresh.HumidityMin,
				HumidityMax    : thresh.HumidityMax,
			},
		}
		datas = append(datas, dd)
	}
	body, err := json.Marshal(datas)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(body)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func GetDeviceData(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	device := ps["device"]
	projectId := ps["projectId"]
	startAt := req.URL.Query().Get("startAt")
	endAt := req.URL.Query().Get("endAt")
	// startAt, endAt like '2019-08-17T06:40:27.995Z'
	var tEnd time.Time
	tStart, err := time.Parse(time.RFC3339, startAt)
	if err == nil {
		tEnd, err = time.Parse(time.RFC3339, endAt)
	}
	if err != nil || tEnd.Before(tStart) {
		strErr := fmt.Sprintf("Invalid time params, startAt:%s, endAt:%s.", startAt, endAt)
		logs.Error(strErr)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strErr))
		return
	}
	datas, err := influxdb.GetDataByTime("temperature", "", startAt, endAt, device, projectId)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	list := influxdb.OutDataList{
		Datas: datas,
		Count: len(datas),
	}
	body, err := json.Marshal(list)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(body)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func ListDevices(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	devices, err := influxdb.GetDevicesByThing("temperature", "", projectId)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("thing id not found"))
		return
	}
	list := influxdb.DeviceList{
		Devices: devices,
	}
	body, err := json.Marshal(list)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(body)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
