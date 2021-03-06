package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
)

type ComponentDetail struct {
	Id            string `orm:"size(64);"`
	ComponentId   string `orm:"size(64);pk"`
	UpdateStatus  int
	Data          string `orm:"type(text);null"`
	UpdateData    string `orm:"type(text);null"`
}

func init() {
	orm.RegisterModel(new(ComponentDetail))
}


func CreateComponentDetail(detail ComponentDetail) string {
	o := orm.NewOrm()
	u2 := uuid.NewV4()
	detail.Id = u2.String()
	// insert
	_, err := o.Insert(&detail)
	if err != nil {
		logs.Error("create detail fail.detail: %v", detail)
		return ""
	}
	logs.Info("create detail id: %v", detail.Id)
	return detail.Id
}

func DeleteComponentDetail(id string) error {
	o := orm.NewOrm()
	b := ComponentDetail{Id: id}
	if _, err := o.Delete(&b); err != nil {
		return err
	}
	logs.Info("delete detail: %v", id)
	return nil
}

func UpdateComponentDetail(detail ComponentDetail) error {
	o := orm.NewOrm()
	if _, err := o.Update(&detail, "update_status", "data", "update_data"); err != nil {
		logs.Error("update detail(%s) fail, err:%s", detail.Id, err.Error())
		return err
	}

	logs.Info("update detail success, id:%v", detail.Id)
	return nil
}

func UpdateDetailUpdateDataAndStatus(detail ComponentDetail) error {
	o := orm.NewOrm()
	if _, err := o.Update(&detail, "update_status", "update_data"); err != nil {
		logs.Error("update detail(%s) update_data fail, err:%s", detail.Id, err.Error())
		return err
	}

	logs.Info("update detail success, id:%v", detail.Id)
	return nil
}
func UpdateDetailStatusOnly(detail ComponentDetail) error {
	o := orm.NewOrm()
	if _, err := o.Update(&detail, "update_status"); err != nil {
		logs.Error("update detail(%s) status fail, err:%s", detail.Id, err.Error())
		return err
	}

	logs.Info("update detail success, id:%v", detail.Id)
	return nil
}

func QueryComponentDetailByComponentId(componentId string) (u *ComponentDetail, err error) {
	o := orm.NewOrm()
	component := ComponentDetail{ComponentId: componentId}
	if err := o.Read(&component); err != nil {
		logs.Error("query component detail(%s) fail: %v", componentId, err.Error())
		return nil, err
	}
	return &component, nil
}


