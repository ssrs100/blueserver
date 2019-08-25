package main

import (
	"fmt"
	"github.com/dimfeld/httptreemux"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/controller"
	"github.com/ssrs100/blueserver/controller/aws"
	"github.com/ssrs100/blueserver/influxdb"
	"github.com/ssrs100/blueserver/mqttclient"
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

func Health(w http.ResponseWriter, req *http.Request, _ map[string]string) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) RegisterRoutes() *httptreemux.TreeMux {
	logs.Debug("Setting route info...")

	// Set the router.
	router := httptreemux.New()

	// Set router options.
	router.PanicHandler = httptreemux.SimplePanicHandler
	router.RedirectTrailingSlash = true

	// Set the routes for the application.

	// Route for health check
	router.GET("/v1/heart", Health)

	// Routes for users
	router.GET("/v1/users", controller.GetUsers)
	router.GET("/v1/users/:projectId", controller.GetUser)
	router.POST("/v1/users", controller.CreateUser)
	router.POST("/v1/users/login", controller.UserLogin)
	router.DELETE("/v1/users/:projectId", controller.DeleteUser)

	// Routes for beacons
	router.POST("/proximity/v1/:projectId/beacons", controller.RegisterBeacon)
	router.GET("/proximity/v1/:projectId/beacons", controller.ListBeacons)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId", controller.DeleteBeacon)
	router.PUT("/proximity/v1/:projectId/beacons/:beaconId", controller.UpdateBeacon)
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/active", controller.ActiveBeancon)
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/deactive", controller.DeActiveBeancon)

	// Routes for attachments
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/attachments", controller.CreateAttachment)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId/attachments/:attachmentId", controller.DeleteAttachment)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId/attachments", controller.DeleteAttachmentByBeacon)
	router.GET("/proximity/v1/:projectId/beacons/:beaconId/attachments", controller.GetAttachmentByBeacon)

	// Routes for hybrid
	router.POST("/proximity/v1/:projectId/getforobserved", controller.GetForObserved)

	// Routes for components
	router.POST("/equipment/v1/:projectId/components", controller.RegisterComponent)
	router.GET("/equipment/v1/:projectId/components", controller.ListComponents)
	router.DELETE("/equipment/v1/:projectId/components/:componentId", controller.DeleteComponent)
	router.PUT("/equipment/v1/:projectId/components/:componentId", controller.UpdateComponent)
	router.GET("/equipment/v1/:projectId/components/:componentId/collections", controller.ListCollections)

	// Routes for component detail
	router.GET("/equipment/v1/:projectId/components/:componentId/detail", controller.GetComponentDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail", controller.UpdateComponentDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail/cancel-modifying", controller.CancelUpdateDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail/sync", controller.SyncComponentDetail)

	router.GET("/aws/v1/:projectId/things", aws.ListThings)
	router.GET("/aws/v1/:projectId/things/:thingName/latest", aws.GetThingLatestData)
	router.GET("/aws/v1/:projectId/things/:thingName/range", aws.GetThingData)
	// Routes for attachments
	router.GET("/app/resource", controller.GetAdPic)
	return router
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
