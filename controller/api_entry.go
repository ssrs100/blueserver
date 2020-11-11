package controller

import (
	"github.com/dimfeld/httptreemux"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/controller/aws"
	"github.com/ssrs100/blueserver/controller/middleware"
	"net/http"
	"runtime"
)

func Health(w http.ResponseWriter, req *http.Request, _ map[string]string) {
	w.WriteHeader(http.StatusOK)
}

func panicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	logs.Debug("==> %s\n", string(buf[:n]))
}

func LoadApi() *httptreemux.TreeMux {
	// Set the router.
	router := httptreemux.New()

	// Set router options.
	router.PanicHandler = panicHandler
	router.RedirectTrailingSlash = true

	// Set the routes for the application.
	s := middleware.NewStack()
	s.Use(middleware.Auth)
	// Route for health check
	router.GET("/v1/heart", Health)

	// Routes for users
	router.GET("/active", ActiveUser)
	router.POST("/v1/users/verify", SendVerifyCode)
	router.POST("/v1/users/password/reset", s.Wrap(ResetPwd))
	router.GET("/v1/users", s.Wrap(GetUsers))
	router.GET("/v1/users/:projectId", s.Wrap(GetUser))
	router.POST("/v1/users", CreateUser)
	router.POST("/v1/users/login", UserLogin)
	router.DELETE("/v1/users/:projectId", s.Wrap(DeleteUser))
	router.POST("/v1/users/:projectId", s.Wrap(BindAwsUser))

	// Routes for beacons
	router.POST("/proximity/v1/:projectId/beacons", RegisterBeacon)
	router.GET("/proximity/v1/:projectId/beacons", ListBeacons)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId", DeleteBeacon)
	router.PUT("/proximity/v1/:projectId/beacons/:beaconId", UpdateBeacon)
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/active", ActiveBeancon)
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/deactive", DeActiveBeancon)

	// Routes for attachments
	router.POST("/proximity/v1/:projectId/beacons/:beaconId/attachments", CreateAttachment)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId/attachments/:attachmentId", DeleteAttachment)
	router.DELETE("/proximity/v1/:projectId/beacons/:beaconId/attachments", DeleteAttachmentByBeacon)
	router.GET("/proximity/v1/:projectId/beacons/:beaconId/attachments", GetAttachmentByBeacon)

	// Routes for hybrid
	router.POST("/proximity/v1/:projectId/getforobserved", GetForObserved)

	// Routes for components
	router.POST("/equipment/v1/:projectId/components", RegisterComponent)
	router.GET("/equipment/v1/:projectId/components", ListComponents)
	router.DELETE("/equipment/v1/:projectId/components/:componentId", DeleteComponent)
	router.PUT("/equipment/v1/:projectId/components/:componentId", UpdateComponent)
	router.GET("/equipment/v1/:projectId/components/:componentId/collections", ListCollections)

	// Routes for component detail
	router.GET("/equipment/v1/:projectId/components/:componentId/detail", GetComponentDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail", UpdateComponentDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail/cancel-modifying", CancelUpdateDetail)
	router.PUT("/equipment/v1/:projectId/components/:componentId/detail/sync", SyncComponentDetail)

	// AWS
	router.GET("/aws/v1/:projectId/things", s.Wrap(aws.ListThings))
	router.POST("/aws/v1/:projectId/things", s.Wrap(aws.RegisterThing))
	router.DELETE("/aws/v1/:projectId/things/:thingName", s.Wrap(aws.RemoveThing))
	router.PUT("/aws/v1/:projectId/things/:thingName", s.Wrap(aws.UpdateThing))
	router.GET("/aws/v2/:projectId/things", s.Wrap(aws.ListThingsV2))
	router.GET("/aws/v1/:projectId/things/:thingName/latest", s.Wrap(aws.GetThingLatestData))
	router.GET("/aws/v1/:projectId/things/:thingName/range-data", s.Wrap(aws.GetThingData))
	router.GET("/aws/v1/:projectId/things/:thingName/device", s.Wrap(aws.GetThingDevice))

	router.GET("/aws/v1/:projectId/devices", s.Wrap(aws.ListDevices))
	router.GET("/aws/v1/:projectId/devices/:device/latest", s.Wrap(aws.GetDeviceLatestData))
	router.GET("/aws/v1/:projectId/devices/latest", s.Wrap(aws.GetMultiDeviceLatestData))
	router.GET("/aws/v1/:projectId/devices/:device/range-data", s.Wrap(aws.GetDeviceData))

	router.GET("/aws/v1/:projectId/devices/:device/thresh", s.Wrap(aws.GetDeviceThresh))
	router.PUT("/aws/v1/:projectId/devices/:device/thresh", s.Wrap(aws.PutDeviceThresh))

	router.GET("/aws/v1/:projectId/notify", s.Wrap(aws.GetUserNotify))
	router.PUT("/aws/v1/:projectId/notify", s.Wrap(aws.AddUserNotify))
	router.DELETE("/aws/v1/:projectId/notify/:subscribeId", s.Wrap(aws.RmvUserNotify))
	// cert
	router.POST("/aws/v1/:projectId/certificate", s.Wrap(aws.UpdateThingCert))

	// Routes for attachments
	router.POST("/app/v1/:projectId/register/dev-token", s.Wrap(RegisterDevToken))

	router.GET("/app/resource", GetAdPic)
	return router
}
