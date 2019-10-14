package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
	"time"
)

const (
	thingTable = "thing"
)

type Thing struct {
	Id        string     `orm:"size(64);pk"`
	Name      string     `orm:"size(128)"`
	AwsName   string     `orm:"size(128)"`
	AwsThing  string     `orm:"size(128)"`
	ProjectId string     `orm:"size(64)"`
	CreateAt  *time.Time `orm:"auto_now_add;type(datetime)"`
}

func init() {
	orm.RegisterModel(new(Thing))
}

func SaveThing(t Thing) error {
	o := orm.NewOrm()
	u2, err := uuid.NewV4()
	if err != nil {
		logs.Error("save dev uuid wrong: %s", err.Error())
		return err
	}
	t.Id = u2.String()
	// insert
	_, err = o.Insert(&t)
	if err != nil {
		logs.Error("save thing fail.thing: %v", t)
		return err
	}
	logs.Info("save thing id: %v", t.Id)
	return nil
}

func DeleteThing(id string) error {
	o := orm.NewOrm()
	b := Thing{Id: id}
	if _, err := o.Delete(&b); err != nil {
		return err
	}
	logs.Info("delete thing: %v", id)
	return nil
}

func GetThing(projectId, name string) *Thing {
	o := orm.NewOrm()
	qs := o.QueryTable(thingTable)
	qs = qs.Filter("project_id", projectId)
	qs = qs.Filter("name", name)

	var thing Thing
	err := qs.One(&thing)
	if err != nil {
		logs.Info("query things fail, err:%s", err.Error())
		return nil
	}
	return &thing
}

func QueryThings(params map[string]interface{}) []*Thing {
	var things []*Thing
	o := orm.NewOrm()
	qs := o.QueryTable(thingTable)

	if projectId, ok := params["project_id"]; ok {
		qs = qs.Filter("project_id", projectId)
	}

	if name, ok := params["name"]; ok {
		qs = qs.Filter("name", name)
	}

	if offset, ok := params["offset"]; ok {
		qs = qs.Limit(offset.(int))
	}

	if limit, ok := params["limit"]; ok {
		qs = qs.Limit(limit.(int))
	}

	qs = qs.OrderBy("create_at")
	_, err := qs.All(&things)
	if err != nil {
		logs.Error("query things fail, err:%s", err.Error())
	}
	return things
}
