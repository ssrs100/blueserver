package main

import (
	"fmt"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/julienschmidt/httprouter"
	"github.com/ssrs100/buleserver1/awsmqtt"
	bluedb2 "github.com/ssrs100/buleserver1/bluedb"
	controller2 "github.com/ssrs100/buleserver1/controller"
	aws2 "github.com/ssrs100/buleserver1/controller/aws"
	mqttclient2 "github.com/ssrs100/buleserver1/mqttclient"
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
	configure *conf.Config
}

var s Server

func Health(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) RegisterRoutes() *httprouter.Router {
	logs.Debug("Setting route info...")

	// Set the router.
	router := httprouter.New()

	// Set router options.
	router.HandleMethodNotAllowed = true
	router.HandleOPTIONS = true
	router.RedirectTrailingSlash = true

	// Set the routes for the application.

	// Route for health check
	router.GET("/v1/heart", Health)

	// Routes for users
	router.GET("/v1/users", controller2.GetUsers)
	router.GET("/v1/users/:projectId", controller2.GetUser)
	router.POST("/v1/users", controller2.CreateUser)
	router.POST("/v1/users/login", controller2.UserLogin)
	router.DELETE("/v1/users/:projectId", controller2.DeleteUser)

	// Routes for beacons
	router.POST("/proximity/v1/:projectId/beacons", controller2.RegisterBeacon)
	router.GET("/proximity/v1/:projectId/beacons", controller2.ListBeacons)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId", controller2.DeleteBeacon)
	router.PUT("/proximity/v1/:projectId/beacons/:beaconId", controller2.UpdateBeacon)
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/active", controller2.ActiveBeancon)
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/deactive", controller2.DeActiveBeancon)

	// Routes for attachments
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/attachments", controller2.CreateAttachment)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId/attachments/:attachmentId", controller2.DeleteAttachment)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId/attachments", controller2.DeleteAttachmentByBeacon)
	router.GET("/proximity/v1/:projectId/beacons/:beaconId/attachments", controller2.GetAttachmentByBeacon)

	// Routes for hybrid
	router.POST("/proximity/v1/:projectId/getforobserved", controller2.GetForObserved)

	// Routes for components
	router.POST("/equipment/v1/:projectId/components", controller2.RegisterComponent)
	router.GET("/equipment/v1/:projectId/components", controller2.ListComponents)
	router.DELETE("/equipment/v1/:projectId/components/:componentId", controller2.DeleteComponent)
	router.PUT("/equipment/v1/:projectId/components/:componentId", controller2.UpdateComponent)
	router.GET("/equipment/v1/:projectId/components/:componentId/collections", controller2.ListCollections)

	// Routes for component detail
	router.GET("/equipment/v1/:projectId/components/:componentId/detail", controller2.GetComponentDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail", controller2.UpdateComponentDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail/cancel-modifying", controller2.CancelUpdateDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail/sync", controller2.SyncComponentDetail)

	router.GET("/aws/v1/:projectId/things", aws2.ListThings)
	return router
}

var stop = make(chan os.Signal)

// Start sets up and starts the main server application
func Start() error {
	logs.InitLog()
	logs.Info("Setting up server...")

	basedir := common.GetAppBaseDir()
	if len(basedir) == 0 {
		logs.Error("Evironment APP_BASE_DIR(app installed root path) should be set.")
		os.Exit(1)
	}

	//获取配置信息
	appConfig := filepath.Join(basedir, "conf", server_config)
	s.configure = conf.LoadFile(appConfig)
	if s.configure == nil {
		errStr := fmt.Sprintf("Can not load %s.", server_config)
		logs.Error(errStr)
		os.Exit(1)
	}

	err := bluedb2.InitDB(s.configure.GetString("db_host"), s.configure.GetInt("db_port"))
	if err != nil {
		errStr := fmt.Sprintf("Can not init db %s.", err.Error())
		logs.Error(errStr)
		os.Exit(1)
	}

	mc := mqttclient2.InitClient(s.configure)
	mc.Start()

	awsmqtt.InitAwsClient()

	router := s.RegisterRoutes()
	host := s.configure.GetString("host")
	port := s.configure.GetInt("port")
	server := &http.Server{Addr: host + ":" + strconv.Itoa(port), Handler: router}

	logs.Debug("Starting server on port %d", port)

	certPath := filepath.Join(basedir, "conf", s.configure.GetString("cert"))
	keyPath := filepath.Join(basedir, "conf", s.configure.GetString("key"))

	err = server.ListenAndServeTLS(certPath, keyPath)
	if err != nil {
		logs.Fatal("ListenAndServeTLS: ", err)
	}

	return nil
}

func main() {
	Start()
}
