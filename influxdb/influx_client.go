package influxdb

import (
	"fmt"
	client "github.com/influxdata/influxdb1-client"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"net/url"
	"strings"
	"time"
)

type ReportData struct {
	Device      string `json:"device"`
	Timestamp   int64  `json:"timestamp"`
	Rssi        int    `json:"rssi"`
	Temperature int    `json:"temperature"`
	Humidity    int    `json:"humidity"`
}

type InfluxClient struct {
	c *client.Client
}

var influx InfluxClient

const (
	dbName    = "blue"
	retention = "default"
)

var columns = []string{"time", "device", "humidity", "rssi", "temperature"}
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

func GetLatest(table string, device string) (data *ReportData, err error) {
	q := client.Query{
		Command: fmt.Sprintf("select %s from %s where device='%s' "+
			"order by time desc limit 1", columnStr, table, device),
		Database: dbName,
	}
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
			if len(data) < len(columns) {
				logs.Warn("columns less %d", len(columns))
				continue
			}
			logs.Debug("%v", data)
			ret := ReportData{}
			ret.Timestamp, _ = data[0].(int64)
			ret.Device, _ = data[1].(string)
			ret.Humidity, _ = data[2].(int)
			ret.Rssi, _ = data[3].(int)
			ret.Temperature, _ = data[4].(int)
			return &ret, nil
		}
	}

	return nil, response.Err
}

func GetDataByTime(table string, device, startAt, endAt string) (fields map[string]interface{}, err error) {
	q := client.Query{
		Command: fmt.Sprintf("select %s from %s where time >= '%s' and time < '%s' device='%s' "+
			"order by time desc", columnStr, table, startAt, endAt, device),
		Database: dbName,
	}
	response, err := influx.c.Query(q)
	if err != nil {
		return nil, err
	}
	for _, v := range response.Results {
		logs.Info("%v", v.Series[0].Values[0])
		bs, err := v.MarshalJSON()
		if err != nil {
			logs.Info("%s", err.Error())
			continue
		}
		logs.Info("%s", string(bs))
	}

	return nil, nil
}
