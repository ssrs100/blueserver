package main

import (
	"fmt"
	client "github.com/influxdata/influxdb1-client"
	"net/url"
)
func main() {
	host, err := url.Parse(fmt.Sprintf("http://%s:%d", "54.183.122.135", 8086))
	if err != nil {
		fmt.Println(err.Error())
	}

	// NOTE: this assumes you've setup a user and have setup shell env variables,
	// namely INFLUX_USER/INFLUX_PWD. If not just omit Username/Password below.
	conf := client.Config{
		URL:      *host,
	}
	con, err := client.NewClient(conf)
	if err != nil {
		fmt.Println(err.Error())
	}
	dur, ver, err := con.Ping()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Happy as a hippo! %v, %s", dur, ver)
}
