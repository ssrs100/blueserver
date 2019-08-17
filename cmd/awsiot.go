package main

import (
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/awsmqtt"
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

	awsmqtt.InitAwsClient()
}
