package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
	"time"
)

type Component struct {
	Id                string    `orm:"size(64);pk"`
	MacAddr           string    `orm:"size(64)"`
	GwMacAddr         string    `orm:"size(64)"`
	Type              string    `orm:"size(64)"`
	ProjectId         string    `orm:"size(64)"`
	Name              string    `orm:"size(64);null"`
	ComponentPassword string    `orm:"size(64)"`
	CreateAt          time.Time `orm:"auto_now_add;type(datetime)"`
}

func init() {
	orm.RegisterModel(new(Component))
}


func CreateComponent(component Component) string {
	o := orm.NewOrm()
	u2, err := uuid.NewV4()
	if err != nil {
		logs.Error("create component uuid wrong: %s", err.Error())
		return ""
	}
	component.Id = u2.String()
	// insert
	_, err = o.Insert(&component)
	if err != nil {
		logs.Error("create component fail.component: %v", err.Error())
		return ""
	}
	logs.Info("create component id: %v", component.Id)
	return component.Id
}

func DeleteComponent(id string) error {
	o := orm.NewOrm()
	b := Component{Id: id}
	if _, err := o.Delete(&b); err != nil {
		return err
	}
	logs.Info("delete component: %v", id)
	return nil
}

func UpdateComponent(com Component) error {
	o := orm.NewOrm()
	if _, err := o.Update(&com, "name", "component_password"); err != nil {
		logs.Error("update component(%s) fail, err:%s", com.Id, err.Error())
		return err
	}

	logs.Info("update component success, id:%v", com.Id)
	return nil
}

func QueryComponents(params map[string]interface{}) []Component {
	var components []Component
	o := orm.NewOrm()
	qs := o.QueryTable("component")

	if projectId, ok := params["project_id"]; ok {
		qs = qs.Filter("project_id", projectId)
	}

	if devType, ok := params["type"]; ok {
		qs = qs.Filter("type", devType)
	}

	if name, ok := params["name"]; ok {
		qs = qs.Filter("name", name)
	}

	if macAddr, ok := params["mac_addr"]; ok {
		qs = qs.Filter("mac_addr", macAddr)
	}

	if macAddr, ok := params["gw_mac_addr"]; ok {
		qs = qs.Filter("gw_mac_addr", macAddr)
	}

	if offset, ok := params["offset"]; ok {
		qs = qs.Limit(offset.(int))
	}

	if limit, ok := params["limit"]; ok {
		qs = qs.Limit(limit.(int))
	}

	qs = qs.OrderBy("create_at")
	_, err := qs.All(&components)
	if err != nil {
		logs.Error("query components fail, err:%s", err.Error())
	}
	return components
}

func QueryComponentById(id string) (u *Component, err error) {
	o := orm.NewOrm()
	component := Component{Id: id}
	if err := o.Read(&component); err != nil {
		logs.Error("query component(%s) fail: %v", id, err.Error())
		return nil, err
	}
	return &component, nil
}

func QueryComponentByMac(mac string) *Component {
	var components []Component
	o := orm.NewOrm()
	qs := o.QueryTable("component")
	qs = qs.Filter("mac_addr", mac)
	_, err := qs.All(&components)
	if err != nil {
		logs.Error("query components fail, err:%s", err.Error())
		return nil
	}

	if len(components) <= 0 {
		logs.Warn("query components by mac get none, mac:%s", mac)
		return nil
	}

	return &components[0]
}

func QueryComponentByMacAndType(mac, addr_type string) *Component {
	var components []Component
	o := orm.NewOrm()
	qs := o.QueryTable("component")
	qs = qs.Filter("mac_addr", mac)
	qs = qs.Filter("type", addr_type)
	_, err := qs.All(&components)
	if err != nil {
		logs.Error("query components fail, err:%s", err.Error())
		return nil
	}

	if len(components) <= 0 {
		logs.Warn("query components by mac get none, mac:%s", mac)
		return nil
	}

	return &components[0]
}
