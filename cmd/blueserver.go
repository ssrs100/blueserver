package main

import (
	"fmt"
	"github.com/dimfeld/httptreemux"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/controller"
	"github.com/ssrs100/blueserver/influxdb"
	"github.com/ssrs100/blueserver/mqttclient"
	"github.com/ssrs100/blueserver/sesscache"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var (
	server_config = "blueserver.json"
)

// Config struct provides configuration fields for the server.
type Server struct {
}

var s Server

func (s *Server) RegisterRoutes() *httptreemux.TreeMux {
	logs.Debug("Setting route info...")
	return controller.LoadApi()
}

var stop = make(chan os.Signal)

// Start sets up and starts the main server application
func Start() error {
	logs.InitLog()
	logs.Info("Setting up server...")

	basedir := utils.GetBasePath()
	if len(basedir) == 0 {
		logs.Error("Evironment APP_BASE_DIR(app installed root path) should be set.")
		os.Exit(1)
	}

	//获取配置信息
	appConfig := filepath.Join(basedir, "conf", server_config)
	if err := conf.Init(appConfig); err != nil {
		logs.Error("appConfig %s not found", appConfig)
		os.Exit(1)
	}
	err := bluedb.InitDB(conf.GetString("db_host"), conf.GetInt("db_port"))
	if err != nil {
		errStr := fmt.Sprintf("Can not init db %s.", err.Error())
		logs.Error(errStr)
		os.Exit(1)
	}
	sesscache.InitRedis()
	mc := mqttclient.InitClient()
	mc.Start()

	influxdb.InitFlux()

	router := s.RegisterRoutes()
	host := conf.GetString("host")
	port := conf.GetInt("port")
	server := &http.Server{Addr: host + ":" + strconv.Itoa(port), Handler: router}

	logs.Debug("Starting server on port %d", port)

	certPath := filepath.Join(basedir, "conf", conf.GetString("cert"))
	keyPath := filepath.Join(basedir, "conf", conf.GetString("key"))

	err = server.ListenAndServeTLS(certPath, keyPath)
	if err != nil {
		logs.Fatal("ListenAndServeTLS: ", err)
	}

	return nil
}

func main() {
	Start()
}
