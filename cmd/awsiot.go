package main

import (
	"fmt"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/awsmqtt"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/influxdb"
	"os"
	"path/filepath"
)

const (
	awsIotConfig = "awsiot.json"
)

func main() {
	logs.InitLog()
	logs.Info("Setting up awsiot...")
	baseDir := utils.GetBasePath()
	if err := conf.Init(filepath.Join(baseDir, "conf", awsIotConfig)); err != nil {
		logs.Error("%s", err.Error())
		os.Exit(1)
	}
	influxdb.InitFlux()
	err := bluedb.InitDB(conf.GetString("db_host"), conf.GetInt("db_port"))
	if err != nil {
		errStr := fmt.Sprintf("Can not init db %s.", err.Error())
		logs.Error(errStr)
		os.Exit(1)
	}
	awsmqtt.InitSns()
	awsmqtt.InitAwsClient()
}
