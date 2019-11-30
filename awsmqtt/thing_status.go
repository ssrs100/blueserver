package awsmqtt

import (
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/common"
	"github.com/ssrs100/blueserver/sesscache"
	"runtime"
	"strconv"
	"time"
)

const (
	OnLine  = "1"
	OffLine = "0"
)

type thingStatus struct {
	stop chan interface{}
}

func (t *thingStatus) Init(stop chan interface{}) {
	sesscache.InitRedis()
	t.stop = stop
	go t.processOffline()
}

func (t *thingStatus) processOffline() {
	timer := time.NewTicker(time.Second * 90)
	for {
		select {
		case <-timer.C:
			t.process()
		case <-t.stop:
			logs.Info("status timer stopped")
			return
		}
	}
}

func (t *thingStatus) process() {
	defer func() {
		if err := recover(); err != nil {
			logs.Error("%v", err)
			buf := make([]byte, 16384)
			buf = buf[:runtime.Stack(buf, true)]
			logs.Error("=== BEGIN goroutine stack dump ===\n%s\n=== END goroutine stack dump ===", buf)
		}
	}()
	param := make(map[string]interface{})
	param["status"] = 1
	things := bluedb.QueryThings(param)
	for _, t := range things {
		key := common.StatusKey(t.Name)
		status := sesscache.Get(key)
		dbStatus := strconv.Itoa(t.Status)
		if OnLine != status && dbStatus == OnLine {
			t.Status = 0
			if err := bluedb.UpdateThingStatus(*t); err != nil {
				logs.Error("update status fail, err:%s", err.Error())
			}
		}
	}
}

