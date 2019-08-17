package influxdb

import (
	"fmt"
	client "github.com/influxdata/influxdb1-client"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"net/url"
	"time"
)

type InfluxClient struct {
	c *client.Client
}

var influx InfluxClient

const (
	dbName    = "blue"
	retention = "default"
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

func Insert(table string, fields map[string]interface{}, rTime *time.Time) error {
	pts := make([]client.Point, 0)
	p := client.Point{
		Measurement: table,
		Fields:      fields,
		Time:        *rTime,
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

func GetLatest(table string, device string) (fields map[string]interface{}, err error) {
	q := client.Query{
		Command: fmt.Sprintf("select time,device,humidity,rssi,temperature from %s where device='%s' "+
			"order by time desc limit 2", table, device),
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

func GetDataByTime(table string, device, startAt, endAt string) (fields map[string]interface{}, err error) {
	q := client.Query{
		Command: fmt.Sprintf("select * from %s where time >= '%s' and time < '%s' device='%s' "+
			"order by time desc", table, startAt, endAt, device),
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
