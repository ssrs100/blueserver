package influxdb

import (
	"encoding/json"
	"github.com/jack0liu/logs"
	"strconv"
	"strings"
)

var sensorColumns = []string{
	columnTime,
	columnDevice,
	columnHumidity,
	columnRssi,
	columnTemperature,
	columnThing,
	columnProjectId,
	columnDeviceName,
	columnPower,
}
var broadcastColumns = []string{
	columnTime,
	columnDevice,
	columnRssi,
	columnThing,
	columnProjectId,
	columnDeviceName,
	columnPower,
	columnData,
}
var sensorColumnStr string
var broadcastColumnStr string

type getOneData func(data []interface{}) *OutData

var tableData = make(map[string]getOneData)

func init() {
	sensorColumnStr = strings.Join(sensorColumns, ",")
	broadcastColumnStr = strings.Join(broadcastColumns, ",")

	tableData[TableBroadcast] = getOneBroadcastData
	tableData[TableTemperature] = getOneSensorData
}

func getOneSensorData(data []interface{}) *OutData {
	if len(data) < len(sensorColumns) {
		logs.Warn("columns less %d", len(sensorColumns))
		return nil
	}
	ret := OutData{}
	ret.Timestamp, _ = data[0].(string)
	ret.Device, _ = data[1].(string)

	humi, ok := data[2].(string)
	if ok {
		h := json.Number(humi)
		ret.Humidity = &h
	} else {
		humidityFloat, ok := data[2].(float64)
		if ok {
			h := json.Number(strconv.FormatFloat(humidityFloat, 'G', 5, 64))
			ret.Humidity = &h
		} else {
			humidityInt, ok := data[2].(int)
			if ok {
				h := json.Number(strconv.Itoa(humidityInt))
				ret.Humidity = &h
			} else {
				logs.Error("invalid humidity:%v", data[2])
			}
		}
	}

	rssi, ok := data[3].(string)
	if ok {
		ret.Rssi = json.Number(rssi)
	} else {
		rssiInt, ok := data[3].(int)
		if ok {
			ret.Rssi = json.Number(strconv.Itoa(rssiInt))
		} else {
			rssiFloat, ok := data[3].(float64)
			if ok {
				ret.Rssi = json.Number(strconv.FormatFloat(rssiFloat, 'G', 5, 64))
			} else {
				logs.Error("invalid rssi:%v", data[2])
			}
		}
	}

	temp, ok := data[4].(string)
	if ok {
		h := json.Number(temp)
		ret.Temperature = &h
	} else {
		tempFloat, ok := data[4].(float64)
		if ok {
			h := json.Number(strconv.FormatFloat(tempFloat, 'G', 5, 64))
			ret.Temperature = &h
		} else {
			tempInt, ok := data[4].(int)
			if ok {
				h := json.Number(strconv.Itoa(tempInt))
				ret.Temperature = &h
			} else {
				logs.Error("invalid temper:%v", data[4])
			}
		}
	}
	thingName, _ := data[5].(string)
	thingSegs := strings.Split(thingName, ":")
	ret.Thing = thingSegs[0]
	ret.ProjectId, _ = data[6].(string)
	ret.DeviceName, _ = data[7].(string)
	ret.Power, _ = data[8].(string)
	return &ret
}

func getOneBroadcastData(data []interface{}) *OutData {
	if len(data) < len(broadcastColumns) {
		logs.Warn("columns less %d", len(broadcastColumns))
		return nil
	}
	ret := OutData{}
	ret.Timestamp, _ = data[0].(string)
	ret.Device, _ = data[1].(string)

	rso := data[2]
	rssi, ok := rso.(string)
	if ok {
		ret.Rssi = json.Number(rssi)
	} else {
		rssiInt, ok := rso.(int)
		if ok {
			ret.Rssi = json.Number(strconv.Itoa(rssiInt))
		} else {
			rssiFloat, ok := rso.(float64)
			if ok {
				ret.Rssi = json.Number(strconv.FormatFloat(rssiFloat, 'G', 5, 64))
			} else {
				logs.Error("invalid rssi:%v", rso)
			}
		}
	}

	thingName, _ := data[3].(string)
	thingSegs := strings.Split(thingName, ":")
	ret.Thing = thingSegs[0]
	ret.ProjectId, _ = data[4].(string)
	ret.DeviceName, _ = data[5].(string)
	ret.Power, _ = data[6].(string)
	d := data[7].(string)
	ret.Data = &d
	return &ret
}