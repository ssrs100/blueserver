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
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
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
	go signalHandle()

	awsmqtt.InitAwsClient()
}

func signalHandle() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGUSR1)
	for {
		<-ch
		var buf [4096]byte
		n := runtime.Stack(buf[:], true)
		logs.Error("==> %s\n", string(buf[:n]))
	}
}
