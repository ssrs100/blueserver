package main

import (
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/awsmqtt"
	"os"
	"path/filepath"
)

const (
	awsIotConfig = "awsiot.json"
)
func main() {
	baseDir := utils.GetBasePath()
	if err := conf.Init(filepath.Join(baseDir, "conf", awsIotConfig)); err != nil {
		logs.Error("%s", err.Error())
		os.Exit(1)
	}
	awsmqtt.InitAwsClient()
}