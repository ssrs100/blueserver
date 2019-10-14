package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
)

type Sys struct {
	Name        string `orm:"size(128);pk"`
	Value       string `orm:"size(128)"`
	Description string `orm:"size(256);null"`
}

func init() {
	orm.RegisterModel(new(Sys))
}

func GetSys(key string) string {
	var sys Sys
	o := orm.NewOrm()
	err := o.Raw("SELECT name, value, description FROM sys WHERE name = ?", key).QueryRow(&sys)
	if err != nil {
		logs.Error("get sys err:%s", err.Error())
		return ""
	}
	return sys.Value
}
