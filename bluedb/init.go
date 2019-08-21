package bluedb

import (
	"fmt"
	"github.com/astaxie/beego/orm"
)

func InitDB(host string, port int) error {

	dsn := fmt.Sprintf("blue:blue@123@tcp(%s:%d)/blue?charset=utf8", host, port)
	// set default database
	if err := orm.RegisterDataBase("default", "mysql", dsn, 30); err != nil {
		return err
	}

	// create table
	if err := orm.RunSyncdb("default", false, true); err != nil {
		return err
	}
	return nil
}
