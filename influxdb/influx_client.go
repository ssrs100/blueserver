package influxdb

import (
	"fmt"
	client "github.com/influxdata/influxdb1-client"
	"github.com/jack0liu/conf"
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

func Insert(table string, fields map[string]interface{}) error {
	pts := make([]client.Point, 0)
	p := client.Point{
		Measurement: table,
		Fields:      fields,
		Time:        time.Now(),
		Precision:   "ms",
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
