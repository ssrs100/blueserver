package awsmqtt

import (
	"github.com/jack0liu/logs"
	"net/http"
)

func StartThing(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	thing := ps["thingName"]
	if len(useClientCache) == 0 {
		logs.Error("client not init")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("client not init"))
		return
	}
	for _, cli := range useClientCache {
		cli.startThing(thing)
		break
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
