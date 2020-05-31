package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
	"time"
)

type DevToken struct {
	Id          string `orm:"size(64);pk"`
	UserId      string `orm:"size(128)"`
	DevId       string `orm:"size(128)"`
	DeviceToken string `orm:"size(128)"`
	UpdateAt *time.Time `orm:"type(datetime)"`
}

func init() {
	orm.RegisterModel(new(DevToken))
}

func RegisterDevToken(token DevToken) error {
	o := orm.NewOrm()
	var devs []*DevToken
	qs := o.QueryTable(new(DevToken))
	qs.Filter("user_id", token.UserId)
	qs.Filter("dev_id", token.DevId)
	qs.OrderBy("update_at")
	_, err := qs.All(&devs)
	if err != nil {
		logs.Error("register dev fail:%v", token)
		return err
	}

	now := time.Now()
	for _, d := range devs {
		if now.Sub(*d.UpdateAt) > 24 * time.Hour {
			o.Delete(d)
		} else {
			d.DeviceToken = token.DeviceToken
			d.UpdateAt = &now
			o.Update(d, "device_token", "update_at")
			return nil
		}
	}

	// insert
	token.Id = uuid.NewV4().String()
	token.UpdateAt = &now
	_, err = o.Insert(&token)
	if err != nil {
		logs.Error("insert dev fail:%v", token)
		return err
	}
	return nil
}


func QueryDevToken(userId string) []*DevToken {
	o := orm.NewOrm()
	var devs []*DevToken
	qs := o.QueryTable(new(DevToken))
	qs.Filter("user_id", userId)
	qs.OrderBy("update_at")
	_, err := qs.All(&devs)
	if err != nil {
		logs.Error("query dev token fail, user_id:%s, err:%s", userId, err.Error())
		return nil
	}
	return devs
}
