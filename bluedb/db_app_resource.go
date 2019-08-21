package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
)

const (
	appResTable = "app_resource"
)
type AppResource struct {
	Id           string `orm:"size(64);pk"`
	Endpoint     string `orm:"size(128)"`
	Uri          string `orm:"size(128)"`
	Type         string `orm:"size(128)"`
	ResourceName string `orm:"size(128);null"`
}

func (app *AppResource) TableName() string {
	return appResTable
}

func init() {
	orm.RegisterModel(new(AppResource))
}

func CreateAppResource(res AppResource) string {
	o := orm.NewOrm()
	u2, err := uuid.NewV4()
	if err != nil {
		logs.Error("create user uuid wrong: %s", err.Error())
		return ""
	}
	res.Id = u2.String()
	// insert
	id, err := o.Insert(&res)
	logs.Info("create res id: %v", id)
	logs.Info("create res: %v", res)
	return res.Id
}

func DeleteAppResource(id string) error {
	o := orm.NewOrm()
	u := AppResource{Id: id}
	if _, err := o.Delete(&u); err != nil {
		return err
	}
	logs.Info("delete res: %v", id)
	return nil
}


func QueryResById(id string) (u *AppResource, err error) {
	o := orm.NewOrm()
	res := AppResource{Id: id}
	if err := o.Read(&res); err != nil {
		logs.Error("query res fail: %v", id)
		return &res, err
	}
	return &res, nil
}

func QueryResByType(rsType string) *AppResource {
	var res AppResource
	o := orm.NewOrm()
	qs := o.QueryTable(appResTable)

	qs = qs.Filter("type", rsType)
	err := qs.One(&res)
	if err != nil {
		logs.Error("query beacon fail, err:%s", err.Error())
		return nil
	}
	return &res
}
