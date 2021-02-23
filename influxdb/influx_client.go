package influxdb

import (
	"encoding/json"
	"errors"
	"fmt"
	client "github.com/influxdata/influxdb1-client"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"net/url"
	"time"
)

type ReportData struct {
	ProjectId   string      `json:"project_id"`
	Device      string      `json:"device"`
	Thing       string      `json:"thing"`
	Timestamp   int64       `json:"timestamp"`
	Rssi        json.Number `json:"rssi"`
	Temperature json.Number `json:"temperature"`
	Humidity    json.Number `json:"humidity"`
	DeviceName  string      `json:"device_name"`
	Power       string      `json:"power"`
	DataType    string      `json:"data_type,omitempty"`
	Data        string      `json:"data,omitempty"`
}

type ReportDataList struct {
	Objects []*ReportData `json:"objs"`
}

type RecordData struct {
	ProjectId   string  `json:"project_id"`
	Device      string  `json:"device"`
	Thing       string  `json:"thing"`
	Timestamp   int64   `json:"timestamp"`
	Rssi        float64 `json:"rssi"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	DeviceName  string  `json:"device_name"`
	Power       float64 `json:"power"`
	DataType    string  `json:"data_type,omitempty"`
	Data        string  `json:"data,omitempty"`
}

type OutData struct {
	ProjectId   string       `json:"project_id"`
	Thing       string       `json:"thing"`
	Device      string       `json:"device"`
	Timestamp   string       `json:"timestamp"`
	Rssi        json.Number  `json:"rssi"`
	Temperature *json.Number `json:"temperature,omitempty"`
	Humidity    *json.Number `json:"humidity,omitempty"`
	DeviceName  string       `json:"device_name"`
	Power       string       `json:"power"`
	Data        *string      `json:"data,omitempty"`
}

type GroupData []interface{}

type OutDataList struct {
	Datas []*OutData `json:"datas"`
	Count int        `json:"count"`
}

type DeviceList struct {
	Devices []string `json:"devices"`
}

type InfluxClient struct {
	c *client.Client
}

var influx InfluxClient

const (
	TableTemperature = "temperature"
	TableBroadcast   = "broadcast"

	dbName    = "blue"
	retention = "default"

	columnTime        = "time"
	columnDevice      = "device"
	columnHumidity    = "humidity"
	columnRssi        = "rssi"
	columnTemperature = "temperature"
	columnThing       = "thing"
	columnProjectId   = "project_id"
	columnDeviceName  = "device_name"
	columnPower       = "power"
	columnData        = "data"

	columnMean = "mean"
)

func InitFlux() {
	host := conf.GetStringWithDefault("influx_host", "localhost")
	hostInfo, err := url.Parse(fmt.Sprintf("http://%s:%d", host, 8086))
	if err != nil {
		panic(err)
	}
	con, err := client.NewClient(client.Config{URL: *hostInfo})
	if err != nil {
		panic(err)
	}
	influx.c = con
}

func InsertSensorData(table string, dataList []*RecordData) error {
	if len(dataList) == 0 {
		logs.Debug("no sensor data")
		return nil
	}
	pts := make([]client.Point, 0)
	for _, data := range dataList {
		fields := make(map[string]interface{})
		fields[columnTemperature] = data.Temperature
		fields[columnHumidity] = data.Humidity
		fields[columnRssi] = data.Rssi
		fields[columnDeviceName] = data.DeviceName
		fields[columnPower] = data.Power
		rdTime := time.Unix(0, data.Timestamp*1000000)

		tags := make(map[string]string)
		tags[columnProjectId] = data.ProjectId
		tags[columnThing] = data.Thing
		tags[columnDevice] = data.Device

		p := client.Point{
			Measurement: table,
			Tags:        tags,
			Fields:      fields,
			Time:        rdTime,
			//Precision:   "n",
		}
		pts = append(pts, p)
	}
	logs.Info("write sensor data:%v", pts)
	bps := client.BatchPoints{
		Points:          pts,
		Database:        dbName,
		RetentionPolicy: retention,
	}
	resp, err := influx.c.Write(bps)
	if err != nil {
		return err
	}
	if resp != nil && resp.Err != nil {
		logs.Error("write err:%v", resp.Err)
	} else {
		logs.Info("write success")
	}
	return nil
}

func InsertBeaconData(table string, dataList []*RecordData) error {
	if len(dataList) == 0 {
		logs.Debug("no beacon data")
		return nil
	}
	pts := make([]client.Point, 0)
	for _, data := range dataList {
		fields := make(map[string]interface{})
		fields[columnRssi] = data.Rssi
		fields[columnDeviceName] = data.DeviceName
		fields[columnPower] = data.Power
		fields[columnData] = data.Data
		rdTime := time.Unix(0, data.Timestamp*1000000)

		tags := make(map[string]string)
		tags[columnProjectId] = data.ProjectId
		tags[columnThing] = data.Thing
		tags[columnDevice] = data.Device

		p := client.Point{
			Measurement: table,
			Tags:        tags,
			Fields:      fields,
			Time:        rdTime,
			//Precision:   "n",
		}
		pts = append(pts, p)
	}
	bps := client.BatchPoints{
		Points:          pts,
		Database:        dbName,
		RetentionPolicy: retention,
	}
	_, err := influx.c.Write(bps)
	if err != nil {
		return err
	}
	return nil
}

func getColumnStr(table string) string {
	columnStr := sensorColumnStr
	if table == TableBroadcast {
		columnStr = broadcastColumnStr
	}
	return columnStr
}

func checkTable(table string) error {
	if _, ok := tableData[table]; !ok {
		return errors.New(fmt.Sprintf("no table(%s)", table))
	}
	return nil
}

func GetLatest(table string, thing, device, projectId string) (data *OutData, err error) {
	if err := checkTable(table); err != nil {
		return nil, err
	}
	columnStr := getColumnStr(table)
	cmd := fmt.Sprintf("select %s from %s where project_id='%s'", columnStr, table, projectId)
	tail := " order by time desc limit 1"
	if len(thing) > 0 {
		cmd = cmd + fmt.Sprintf(" and thing='%s'", thing)
	}
	if len(device) > 0 {
		cmd = cmd + fmt.Sprintf(" and device='%s'", device)
	}
	cmd = cmd + tail

	q := client.Query{
		Command:  cmd,
		Database: dbName,
	}
	logs.Debug("%s", q.Command)
	response, err := influx.c.Query(q)
	if err != nil {
		return nil, err
	}
	for _, v := range response.Results {
		if len(v.Series) == 0 {
			logs.Warn("series is 0")
			continue
		}
		for _, data := range v.Series[0].Values {
			return tableData[table](data), nil
		}
	}

	return nil, response.Err
}

func GetGroupDataByTime(measurement string, thing, startAt, endAt, device, projectId, timeInterval string) (datas [][]interface{}, err error) {
	// startAt, endAt like '2019-08-17T06:40:27.995Z'
	if measurement != columnHumidity && measurement != columnTemperature {
		return nil, fmt.Errorf("invalid measurement %s", measurement)
	}

	cmd := fmt.Sprintf("select mean(%s) from temperature where device='%s' and project_id='%s'",
		measurement, device, projectId)
	tail := fmt.Sprintf(" and time >= '%s' and time < '%s' GROUP BY time(%s) fill(linear)", startAt, endAt, timeInterval)
	if len(thing) > 0 {
		cmd = cmd + fmt.Sprintf(" and thing='%s'", thing)
	}
	cmd = cmd + tail
	q := client.Query{
		Command:  cmd,
		Database: dbName,
	}
	logs.Debug("%s", q.Command)
	response, err := influx.c.Query(q)
	if err != nil {
		return nil, err
	}
	retList := make([][]interface{}, 0)
	for _, v := range response.Results {
		if len(v.Series) == 0 {
			logs.Warn("series is 0")
			continue
		}
		logs.Debug("%v", v.Series[0].Values)
		for _, data := range v.Series[0].Values {
			retList = append(retList, data)
		}
	}

	return retList, nil
}

func GetDataByTime(table string, thing, startAt, endAt, device, projectId string) (datas []*OutData, err error) {
	// startAt, endAt like '2019-08-17T06:40:27.995Z'
	if err := checkTable(table); err != nil {
		return nil, err
	}
	columnStr := getColumnStr(table)
	cmd := fmt.Sprintf("select %s from %s where time >= '%s' and time < '%s' and project_id='%s'", columnStr, table, startAt, endAt, projectId)
	tail := " order by time desc limit 1000"
	if len(thing) > 0 {
		cmd = cmd + fmt.Sprintf(" and thing='%s'", thing)
	}
	if len(device) > 0 {
		cmd = cmd + fmt.Sprintf(" and device='%s'", device)
	}
	cmd = cmd + tail
	q := client.Query{
		Command:  cmd,
		Database: dbName,
	}
	logs.Debug("%s", q.Command)
	response, err := influx.c.Query(q)
	if err != nil {
		return nil, err
	}
	retList := make([]*OutData, 0)
	for _, v := range response.Results {
		if len(v.Series) == 0 {
			logs.Warn("series is 0")
			continue
		}
		for _, data := range v.Series[0].Values {
			d := tableData[table](data)
			retList = append(retList, d)
		}
	}

	return retList, nil
}

func GetDevicesByThing(table string, thing, projectId string) (devices []string, err error) {
	//cmd := fmt.Sprintf("select distinct(device) from %s where project_id='%s'", table, projectId)
	if err := checkTable(table); err != nil {
		return nil, err
	}
	cmd := fmt.Sprintf("select count(*) from %s where project_id='%s' group by device", table, projectId)
	if len(thing) > 0 {
		cmd = cmd + fmt.Sprintf(" and thing='%s'", thing)
	}
	var q client.Query
	q = client.Query{
		Command:  cmd,
		Database: dbName,
	}
	logs.Debug("%s", q.Command)
	response, err := influx.c.Query(q)
	if err != nil {
		return nil, err
	}
	retList := make([]string, 0)
	for _, v := range response.Results {
		if len(v.Series) == 0 {
			logs.Warn("series is 0")
			continue
		}
		for _, s := range v.Series {
			for _, data := range s.Tags {
				logs.Debug("%v", data)
				retList = append(retList, data)
			}
		}

	}

	return retList, nil
}

func DeleteData(table string, thing, projectId string) error {
	//cmd := fmt.Sprintf("select distinct(device) from %s where project_id='%s'", table, projectId)
	cmd := fmt.Sprintf("delete from %s where project_id='%s'", table, projectId)
	if len(thing) > 0 {
		cmd = cmd + fmt.Sprintf(" and thing='%s'", thing)
	}
	var q client.Query
	q = client.Query{
		Command:  cmd,
		Database: dbName,
	}
	logs.Debug("%s", q.Command)
	_, err := influx.c.Query(q)
	if err != nil {
		return err
	}
	return nil
}
