package bluedb

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
)

type User struct {
	Id          string `orm:"size(64);pk"`
	Name        string `orm:"size(128)"`
	Passwd      string `orm:"size(128)"`
	Email       string `orm:"size(128);null"`
	Status      int    `orm:"default(0)"`
	Mobile      string `orm:"size(128);null"`
	Address     string `orm:"size(512);null"`
	AwsUsername string `orm:"size(128);null"`
	AccessKey   string `orm:"size(128);null"`
	SecretKey   string `orm:"size(128);null"`
}

func init() {
	orm.RegisterModel(new(User))
}

func CreateUser(user User) string {
	o := orm.NewOrm()
	u2 := uuid.NewV4()
	user.Id = u2.String()
	// insert
	id, err := o.Insert(&user)
	if err != nil {
		logs.Error("create user fail, err:%s", err.Error())
	}
	logs.Info("create user id: %v", id)
	logs.Info("create user: %v", user)
	return user.Id
}

func UpdateUser(user User) error {
	o := orm.NewOrm()
	_, err := o.Update(&user)
	return err
}

func DeleteUser(id string) error {
	o := orm.NewOrm()
	u := User{Id: id}
	if _, err := o.Delete(&u); err != nil {
		return err
	}
	logs.Info("delete user: %v", id)
	return nil
}

func QueryUsers(params map[string]interface{}) []User {
	var users []User
	// 获取 QueryBuilder 对象. 需要指定数据库驱动参数。
	// 第二个返回值是错误对象，在这里略过
	qb, err := orm.NewQueryBuilder("mysql")
	if err != nil {
		logs.Error("build sql error:%s", err.Error())
		return nil
	}

	// 构建查询对象
	qb = qb.Select("*").From("user")
	name, ok := params["name"]
	if ok {
		qb = qb.Where("name like ?")
	}
	qb = qb.OrderBy("name").Desc()
	if limit, ok := params["limit"]; ok {
		qb = qb.Limit(limit.(int))
	}
	if offset, ok := params["offset"]; ok {
		qb = qb.Limit(offset.(int))
	}

	// 导出 SQL 语句
	sql := qb.String()
	logs.Debug(sql)
	// 执行 SQL 语句
	o := orm.NewOrm()
	if name != nil && len(name.(string)) > 0 {
		o.Raw(sql, name).QueryRows(&users)
	} else {
		o.Raw(sql).QueryRows(&users)
	}

	return users
}

func QueryUserById(id string) (u User, err error) {
	o := orm.NewOrm()
	user := User{Id: id}
	if err := o.Read(&user); err != nil {
		logs.Error("query user fail: %v", id)
		return user, err
	}
	return user, nil
}

func QueryUserByEmail(email string) *User {
	var users []User
	// 获取 QueryBuilder 对象. 需要指定数据库驱动参数。
	// 第二个返回值是错误对象，在这里略过
	qb, err := orm.NewQueryBuilder("mysql")
	if err != nil {
		logs.Error("build sql error:%s", err.Error())
		return nil
	}

	// 构建查询对象
	qb = qb.Select("*").From("user").Where("email = ?")

	// 导出 SQL 语句
	sql := qb.String()
	logs.Debug(sql)
	// 执行 SQL 语句
	o := orm.NewOrm()
	o.Raw(sql, email).QueryRows(&users)
	if len(users) > 0 {
		return &users[0]
	} else {
		return nil
	}
}

func QueryUserByName(name string) (*User, error) {
	var users []User
	qb, err := orm.NewQueryBuilder("mysql")
	if err != nil {
		logs.Error("build sql error:%s", err.Error())
		return nil, err
	}

	// 构建查询对象
	qb = qb.Select("*").From("user").Where("name = ?")

	// 导出 SQL 语句
	sql := qb.String()
	logs.Debug(sql)
	// 执行 SQL 语句
	o := orm.NewOrm()
	o.Raw(sql, name).QueryRows(&users)
	if len(users) > 0 {
		return &users[0], nil
	} else {
		return nil, nil
	}
}
