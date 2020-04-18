package common

import (
	"crypto/sha512"
)

const (
	ComponentBeaconType  = "BEACON"
	ComponentGatewayType = "GATEWAY"

	CookieSessionId = "X-SessionID-B"

	XAuthB = "X-Auth-B"

	TestThing = "only_for_test_bs"

	MinTemp = 0
	MaxTemp = 30
	MinHumi = 30
	MaxHumi = 60
)

func GenToken(id, passwd string) string {
	hash := sha512.New()
	return string(hash.Sum([]byte(id + passwd)))
}

func StatusKey(thing string) string {
	return "status_" + thing
}

func NoticeKey(projectId, key string) string {
	return "notice_" + projectId + "_" + key
}
