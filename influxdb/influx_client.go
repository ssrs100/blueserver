package influxdb

import (
	"encoding/json"
	"fmt"
	client "github.com/influxdata/influxdb1-client"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"net/url"
	"strings"
	"time"
)

type ReportData struct {
	Device      string  `json:"device"`
	Thing       string  `json:"thing"`
	Timestamp   int64   `json:"timestamp"`
	Rssi        int     `json:"rssi"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

type OutData struct {
	Thing       string      `json:"thing"`
	Device      string      `json:"device"`
	Timestamp   string      `json:"timestamp"`
	Rssi        json.Number `json:"rssi"`
	Temperature json.Number `json:"temperature"`
	Humidity    json.Number `json:"humidity"`
}

type OutDataList []OutData

type InfluxClient struct {
	c *client.Client
}

var influx InfluxClient

const (
	dbName    = "blue"
	retention = "default"
)

var columns = []string{"time", "device", "humidity", "rssi", "temperature", "thing"}
var columnStr string

func init() {
	for _, v := range columns {
		columnStr = columnStr + v + ","
	}
	columnStr = strings.TrimRight(columnStr, ",")
}

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

func Insert(table string, data *ReportData) error {
	fields := make(map[string]interface{})
	fields["device"] = data.Device
	fields["temperature"] = data.Temperature
	fields["humidity"] = data.Humidity
	fields["rssi"] = data.Rssi
	fields["thing"] = data.Thing
	rdTime := time.Unix(0, data.Timestamp*1000000)

	pts := make([]client.Point, 0)
	p := client.Point{
		Measurement: table,
		Fields:      fields,
		Time:        rdTime,
		//Precision:   "n",
	}
	pts = append(pts, p)

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

func GetLatest(table string, thing string) (data *OutData, err error) {
	q := client.Query{
		Command: fmt.Sprintf("select %s from %s where thing='%s' "+
			"order by time desc limit 1", columnStr, table, thing),
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
			return getOneData(data), nil
		}
	}

	return nil, response.Err
}

func getOneData(data []interface{}) *OutData {
	if len(data) < len(columns) {
		logs.Warn("columns less %d", len(columns))
		return nil
	}
	logs.Debug("%v", data)
	ret := OutData{}
	ret.Timestamp, _ = data[0].(string)
	ret.Device, _ = data[1].(string)
	ret.Humidity, _ = data[2].(json.Number)
	ret.Rssi, _ = data[3].(json.Number)
	ret.Temperature, _ = data[4].(json.Number)
	ret.Thing, _ = data[5].(string)
	return &ret
}

func GetDataByTime(table string, thing, startAt, endAt string) (datas []*OutData, err error) {
	// startAt, endAt like '2019-08-17T06:40:27.995Z'
	q := client.Query{
		Command: fmt.Sprintf("select %s from %s where time >= '%s' and time < '%s' and thing='%s' "+
			"order by time desc limit 1000", columnStr, table, startAt, endAt, thing),
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
			d := getOneData(data)
			retList = append(retList, d)
		}
	}

	return retList, nil
}
