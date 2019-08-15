package bluedb

import (
	"fmt"
	"github.com/ssrs100/orm"
)

func InitDB(host string, port int) error {
	// register model
	orm.RegisterModel(new(User))
	orm.RegisterModel(new(Beacon))
	orm.RegisterModel(new(Attachment))
	orm.RegisterModel(new(Component))
	orm.RegisterModel(new(Collection))
	orm.RegisterModel(new(ComponentDetail))

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
