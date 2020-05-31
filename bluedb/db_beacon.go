package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
	"time"
)

type Beacon struct {
	Id          string    `orm:"size(64);pk"`
	DeviceId    string    `orm:"size(128)"`
	Type        string    `orm:"size(64)"`
	ProjectId   string    `orm:"size(64)"`
	Status      string    `orm:"size(32);null"`
	Description string    `orm:"size(256);null"`
	Create_at   time.Time `orm:"auto_now_add;type(datetime)"`
}

func init() {
	orm.RegisterModel(new(Beacon))
}


func CreateBeacon(beacon Beacon) string {
	o := orm.NewOrm()
	u2 := uuid.NewV4()
	beacon.Id = u2.String()
	// insert
	_, err := o.Insert(&beacon)
	if err != nil {
		logs.Error("create beacon fail.beacon: %v", beacon)
	}
	logs.Info("create beacon id: %v", beacon.Id)
	return beacon.Id
}

func DeleteBeacon(id string) error {
	o := orm.NewOrm()
	b := Beacon{Id: id}
	if _, err := o.Delete(&b); err != nil {
		return err
	}
	logs.Info("delete beacon: %v", id)
	return nil
}

func QueryBeacons(params map[string]interface{}) []Beacon {
	var beacons []Beacon
	o := orm.NewOrm()
	qs := o.QueryTable("beacon")

	if projectId, ok := params["project_id"]; ok {
		qs = qs.Filter("project_id", projectId)
	}

	if devType, ok := params["type"]; ok {
		qs = qs.Filter("type", devType)
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
	_, err := qs.All(&beacons)
	if err != nil {
		logs.Error("query beacon fail, err:%s", err.Error())
	}
	return beacons
}

func QueryBeaconById(id string) (u *Beacon, err error) {
	o := orm.NewOrm()
	beacon := Beacon{Id: id}
	if err := o.Read(&beacon); err != nil {
		logs.Error("query beacon fail: %v", id)
		return nil, err
	}
	return &beacon, nil
}

func UpdateBeacon(beacon Beacon) error {
	o := orm.NewOrm()
	if _, err := o.Update(&beacon, "description"); err != nil {
		logs.Error("update beacon(%s) fail, err:%s", beacon.Id, err.Error())
		return err
	}

	logs.Info("update beacon success, id:%v", beacon.Id)
	return nil
}

func UpdateBeaconStatus(beanconId, status string) error {
	o := orm.NewOrm()
	beacon := Beacon{Id: beanconId, Status: status}
	if _, err := o.Update(&beacon, "status"); err != nil {
		logs.Error("update beacon status(%s) fail, err:%s", status, err.Error())
		return err
	}

	logs.Info("update beacon status success, id:%v, status:%v", beacon.Id, status)
	return nil
}
