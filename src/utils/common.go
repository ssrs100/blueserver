package utils

import (
	"crypto/sha512"
	"os"
)

const (
	APP_BASE_KEY string = "APP_BASE_DIR"
)

const (
	ComponentBeaconType  = "BEACON"
	ComponentGatewayType = "GATEWAY"
)


var (
	appBaseDir string
)

func GetAppBaseDir() string {
	if len(appBaseDir) > 0 {
		return appBaseDir
	}
	appBaseDir = os.Getenv(APP_BASE_KEY)
	return appBaseDir
}


func GenToken(id, passwd string) string {
	hash := sha512.New()
	return string(hash.Sum([]byte(id+passwd)))
}