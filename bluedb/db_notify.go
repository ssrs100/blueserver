package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
)

type Notify struct {
	Id        string `orm:"size(64);pk"`
	ProjectId string `orm:"size(64)"`
	Device    string `orm:"size(128)"`
	Key       string `orm:"size(64)"`
	Cause     string `orm:"size(64)"`
	Noticed   string `orm:"size(32)"`
}

func init() {
	orm.RegisterModel(new(Notify))
}

func SaveNotice(notice Notify) string {
	o := orm.NewOrm()
	u2, err := uuid.NewV4()
	if err != nil {
		logs.Error("create notify uuid wrong: %s", err.Error())
		return ""
	}
	notice.Id = u2.String()
	// insert
	_, err = o.Insert(&notice)
	return notice.Id
}

func DeleteNotice(projectId, device, key string) error {
	o := orm.NewOrm()
	if _, err := o.Raw("delete from notify where project_id=? and device=? and `key`=?", projectId, device, key).Exec(); err != nil {
		return err
	}
	return nil
}

func QueryNoticeByDevice(projectId, device, key string) (u *Notify, err error) {
	var ns []*Notify
	o := orm.NewOrm()
	qs := o.QueryTable("notify")
	qs = qs.Filter("project_id", projectId)
	qs = qs.Filter("device", device)
	qs = qs.Filter("key", key)
	_, err = qs.All(&ns)
	if err != nil {
		logs.Error("query Notice fail, err:%s", err.Error())
		return nil, err
	}

	if len(ns) <= 0 {
		return nil, nil
	}

	return ns[0], nil
}

func QueryNoticeByDeviceWithCause(projectId, device, key, cause string) (u *Notify, err error) {
	var ns []*Notify
	o := orm.NewOrm()
	qs := o.QueryTable("notify")
	qs = qs.Filter("project_id", projectId)
	qs = qs.Filter("device", device)
	qs = qs.Filter("key", key)
	qs = qs.Filter("cause", cause)
	_, err = qs.All(&ns)
	if err != nil {
		logs.Error("query Notice fail, err:%s", err.Error())
		return nil, err
	}

	if len(ns) <= 0 {
		return nil, nil
	}

	return ns[0], nil
}
