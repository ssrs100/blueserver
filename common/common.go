package common

import (
	"crypto/sha512"
)

const (
	ComponentBeaconType  = "BEACON"
	ComponentGatewayType = "GATEWAY"
)

func GenToken(id, passwd string) string {
	hash := sha512.New()
	return string(hash.Sum([]byte(id+passwd)))
}