package bluedb

import (
	"orm"
	"time"
)

type Collection struct {
	Id          string    `orm:"size(64);pk"`
	ComponentId string    `orm:"size(64)"`
	Rssi        int
	Data        string    `orm:"type(text);null"`
	CreateAt    time.Time `orm:"auto_now_add;type(datetime)"`
}

func AddBatchCollections(cols []Collection) error {
	o := orm.NewOrm()
	// insert
	_, err := o.InsertMulti(len(cols), cols)
	if err != nil {
		log.Error("AddBatchCollections fail.cols: %v", cols)
		return err
	}
	log.Debug("add batch collections success.")
	return nil
}

func QueryCollections(params map[string]interface{}) []Collection {
	var collections []Collection
	o := orm.NewOrm()
	//qs := o.QueryTable("collection")
	//
	//if componentId, ok := params["component_id"]; ok {
	//	qs = qs.Filter("component_id", componentId)
	//}
	//
	//if startTime, ok := params["start_time"]; ok {
	//	log.Debug("startTime:%v", startTime)
	//	qs = qs.Filter("create_at__gte", startTime)
	//}
	//
	//if endTime, ok := params["end_time"]; ok {
	//	log.Debug("endTime:%v", endTime)
	//	qs = qs.Filter("create_at__lt", endTime)
	//}
	//qs = qs.OrderBy("create_at")
	//_, err := qs.All(&collections)


	_, err := o.Raw("SELECT * FROM collection WHERE component_id = ? and " +
		"create_at >= ? and create_at < ? order by create_at asc",
		params["component_id"], params["start_time"], params["end_time"]).QueryRows(&collections)
	if err != nil {
		log.Error("query collections fail, err:%s", err.Error())
	}
	return collections
}
