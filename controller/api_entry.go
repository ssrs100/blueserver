package controller

import (
	"github.com/dimfeld/httptreemux"
	"github.com/ssrs100/blueserver/controller/aws"
	"github.com/ssrs100/blueserver/controller/middleware"
	"net/http"
)

func Health(w http.ResponseWriter, req *http.Request, _ map[string]string) {
	w.WriteHeader(http.StatusOK)
}

func LoadApi() *httptreemux.TreeMux {
	// Set the router.
	router := httptreemux.New()

	// Set router options.
	router.PanicHandler = httptreemux.SimplePanicHandler
	router.RedirectTrailingSlash = true

	// Set the routes for the application.
	s := middleware.NewStack()
	s.Use(middleware.PassThrough)
	// Route for health check
	router.GET("/v1/heart", s.Wrap(Health))

	// Routes for users
	router.GET("/v1/users", GetUsers)
	router.GET("/v1/users/:projectId", GetUser)
	router.POST("/v1/users", CreateUser)
	router.POST("/v1/users/login", UserLogin)
	router.DELETE("/v1/users/:projectId", DeleteUser)
	router.POST("/v1/users/:projectId", BindAwsUser)

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

	router.GET("/aws/v1/:projectId/things", aws.ListThings)
	router.GET("/aws/v1/:projectId/things/:thingName/latest", aws.GetThingLatestData)
	router.GET("/aws/v1/:projectId/things/:thingName/range-data", aws.GetThingData)
	// Routes for attachments
	router.GET("/app/resource", GetAdPic)
	return router
}
