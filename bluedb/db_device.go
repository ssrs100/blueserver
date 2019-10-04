package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
	"time"
)

type Device struct {
	Id          string     `orm:"size(64);pk"`
	DeviceId    string     `orm:"size(128)"`
	Thing       string     `orm:"size(128)"`
	ProjectId   string     `orm:"size(64)"`
	Status      string     `orm:"size(32);null"`
	Description string     `orm:"size(256);null"`
	CreateAt    *time.Time `orm:"auto_now_add;type(datetime)"`
}

func init() {
	orm.RegisterModel(new(Device))
}

func SaveDevice(dev Device) string {
	o := orm.NewOrm()
	u2, err := uuid.NewV4()
	if err != nil {
		logs.Error("save dev uuid wrong: %s", err.Error())
		return ""
	}
	dev.Id = u2.String()
	// insert
	_, err = o.Insert(&dev)
	if err != nil {
		logs.Error("save dev fail.dev: %v", dev)
	}
	logs.Info("save dev id: %v", dev.Id)
	return dev.Id
}

func DeleteDevice(id string) error {
	o := orm.NewOrm()
	b := Device{Id: id}
	if _, err := o.Delete(&b); err != nil {
		return err
	}
	logs.Info("delete device: %v", id)
	return nil
}

func QueryDevices(params map[string]interface{}) []Device {
	var devices []Device
	o := orm.NewOrm()
	qs := o.QueryTable("device")

	if projectId, ok := params["project_id"]; ok {
		qs = qs.Filter("project_id", projectId)
	}

	if thing, ok := params["thing"]; ok {
		qs = qs.Filter("thing", thing)
	}
	if deviceId, ok := params["device_id"]; ok {
		qs = qs.Filter("device_id", deviceId)
	}

	if status, ok := params["status"]; ok {
		qs = qs.Filter("status", status)
	}

	if offset, ok := params["offset"]; ok {
		qs = qs.Limit(offset.(int))
	}

	if limit, ok := params["limit"]; ok {
		qs = qs.Limit(limit.(int))
	}

	qs = qs.OrderBy("create_at")
	_, err := qs.All(&devices)
	if err != nil {
		logs.Error("query devices fail, err:%s", err.Error())
	}
	return devices
}
