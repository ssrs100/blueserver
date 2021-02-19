package influxdb

import (
	"encoding/json"
	"github.com/jack0liu/logs"
	"reflect"
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

func toString(val interface{}) string {
	switch val.(type) {
	case json.Number:
		v := val.(json.Number)
		return string(v)
	case string:
		logs.Debug("this is string, %v", val)
		return val.(string)
	case float64:
		v := val.(float64)
		logs.Debug("this is float64, %v", val)
		return strconv.FormatFloat(v, 'G', 5, 64)
	case float32:
		v := val.(float32)
		logs.Debug("this is float32, %v", val)
		return strconv.FormatFloat(float64(v), 'G', 5, 64)
	case int:
		v := val.(int)
		logs.Debug("this is int, %v", val)
		return strconv.Itoa(v)
	default:
		logs.Error("unknown data type, %v", reflect.TypeOf(val))
		return ""
	}
}

func getOneSensorData(data []interface{}) *OutData {
	if len(data) < len(sensorColumns) {
		logs.Warn("columns less %d", len(sensorColumns))
		return nil
	}
	ret := OutData{}
	ret.Timestamp, _ = data[0].(string)
	ret.Device, _ = data[1].(string)

	humi := json.Number(toString(data[2]))
	ret.Humidity = &humi

	rssi := json.Number(toString(data[3]))
	ret.Rssi = rssi

	temp := json.Number(toString(data[4]))
	ret.Temperature = &temp

	power := toString(data[8])
	ret.Power = power + "%"

	thingName, _ := data[5].(string)
	thingSegs := strings.Split(thingName, ":")
	ret.Thing = thingSegs[0]
	ret.ProjectId, _ = data[6].(string)
	ret.DeviceName, _ = data[7].(string)
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

	rssi := json.Number(toString(data[2]))
	ret.Rssi = rssi

	power := toString(data[8])
	ret.Power = power + "%"

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