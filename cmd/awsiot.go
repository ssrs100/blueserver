package main

import (
	"fmt"
	"github.com/dimfeld/httptreemux"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/awsmqtt"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/influxdb"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
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
	go startHttp()
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

func panicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	logs.Debug("==> %s\n", string(buf[:n]))
}

func startHttp() {
	router := httptreemux.New()

	// Set router options.
	router.PanicHandler = panicHandler
	router.RedirectTrailingSlash = true

	// Set the routes for the application.
	// Route for health check
	router.POST("/v1/things/:thingName/start", awsmqtt.StartThing)

	host := conf.GetString("host")
	port := conf.GetInt("http_port")
	server := &http.Server{Addr: host + ":" + strconv.Itoa(port), Handler: router}

	logs.Debug("Starting http server on port %d", port)

	err := server.ListenAndServe()
	if err != nil {
		logs.Error("ListenAndServe err: ", err)
	}
}
