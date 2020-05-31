package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
)

type DeviceThresh struct {
	Id             string  `orm:"size(64);pk"`
	DevId          string  `orm:"size(64)"` // db in device
	ProjectId      string  `orm:"size(64)"`
	DeviceId       string  `orm:"size(128)"`
	TemperatureMin float32 `orm:"default(0)"`
	HumidityMin    float32 `orm:"default(0)"`
	TemperatureMax float32 `orm:"default(0)"`
	HumidityMax    float32 `orm:"default(0)"`
}

func init() {
	orm.RegisterModel(new(DeviceThresh))
}

func SaveDevThresh(dev DeviceThresh) error {
	o := orm.NewOrm()
	u2 := uuid.NewV4()
	dev.Id = u2.String()
	// insert
	_, err := o.Insert(&dev)
	if err != nil {
		logs.Error("save dev thresh fail.dev: %v", dev)
		return err
	}
	logs.Info("save dev thresh id: %v", dev.Id)
	return nil
}

func UpdateDevThresh(dev DeviceThresh) error {
	o := orm.NewOrm()
	// update
	_, err := o.Update(&dev, "temperature_min", "humidity_min", "temperature_max", "humidity_max")
	if err != nil {
		logs.Error("update dev thresh fail.dev: %v", dev)
		return err
	}
	logs.Info("update dev thresh success")
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
	return nil, nil
}
