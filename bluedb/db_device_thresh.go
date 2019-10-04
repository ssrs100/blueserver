package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
)

type DeviceThresh struct {
	Id             string `orm:"size(64);pk"`
	DevId          string `orm:"size(64)"` // db in device
	DeviceId       string `orm:"size(128)"`
	TemperatureMin int    `orm:"default(0)"`
	HumidityMin    int    `orm:"default(0)"`
	TemperatureMax int    `orm:"default(0)"`
	HumidityMax    int    `orm:"default(0)"`
}

func init() {
	orm.RegisterModel(new(DeviceThresh))
}

func SaveDevThresh(dev DeviceThresh) error {
	o := orm.NewOrm()
	u2, err := uuid.NewV4()
	if err != nil {
		logs.Error("save dev uuid wrong: %s", err.Error())
		return err
	}
	dev.Id = u2.String()
	// insert
	_, err = o.Insert(&dev)
	if err != nil {
		logs.Error("save dev thresh fail.dev: %v", dev)
		return err
	}
	logs.Info("save dev thresh id: %v", dev.Id)
	return nil
}

func DeleteDevThresh(id string) error {
	o := orm.NewOrm()
	b := DeviceThresh{Id: id}
	if _, err := o.Delete(&b); err != nil {
		return err
	}
	logs.Info("delete device thresh: %v", id)
	return nil
}

func QueryDevThresh(projectId, deviceId string) (*DeviceThresh, error) {
	var devices []*DeviceThresh
	o := orm.NewOrm()
	qs := o.QueryTable("device_thresh")

	qs = qs.Filter("project_id", projectId)
	qs = qs.Filter("device_id", deviceId)
	_, err := qs.All(&devices)
	if err != nil {
		logs.Error("query devices fail, err:%s", err.Error())
		return nil, err
	}
	if len(devices) > 0 {
		return devices[0], nil
	}
	return nil, orm.ErrNoRows
}
