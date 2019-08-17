package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
)

type Notify struct {
	Id      string `orm:"size(64);pk"`
	Device  string `orm:"size(128)"`
	Noticed string `orm:"size(32)"`
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

func DeleteNotice(device string) error {
	o := orm.NewOrm()
	if _, err := o.Raw("delete from notify where device=?", device).Exec(); err != nil {
		return err
	}
	return nil
}

func QueryNoticeByDevice(device string) (u *Notify, err error) {
	var ns []*Notify
	o := orm.NewOrm()
	qs := o.QueryTable("notify")
	qs = qs.Filter("device", device)
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
