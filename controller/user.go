package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	utils "github.com/ssrs100/blueserver/common"
	"github.com/ssrs100/blueserver/controller/aws"
	"github.com/ssrs100/blueserver/mqttclient"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type User struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Passwd  string `json:"passwd"`
	Email   string `json:"email"`
	Mobile  string `json:"mobile"`
	Address string `json:"address"`
}

type BindAwsUserReq struct {
	Name      string `json:"aws_username"`
	AccessKey string `json:"aws_access_key"`
	SecretKey string `json:"aws_secret_key"`
}

func (u *User) dbObjectTrans(beacon bluedb.User) User {
	u1 := User{
		Id:      beacon.Id,
		Name:    beacon.Name,
		Passwd:  "******",
		Email:   beacon.Email,
		Mobile:  beacon.Mobile,
		Address: beacon.Address,
	}
	return u1
}

func (u *User) dbListObjectTrans(users []bluedb.User) []User {
	ret := make([]User, 0)
	for _, v := range users {
		ret = append(ret, u.dbObjectTrans(v))
	}
	return ret
}

func GetUsers(w http.ResponseWriter, req *http.Request, _ map[string]string) {
	req.ParseForm()
	params := make(map[string]interface{})
	if limit, err := strconv.Atoi(req.Form.Get("limit")); err == nil {
		params["limit"] = limit
	}

	if offset, err := strconv.Atoi(req.Form.Get("offset")); err == nil {
		params["offset"] = offset
	}
	name := req.Form.Get("name")
	if len(name) > 0 {
		params["name"] = name
	}

	logs.Debug("params:%v", params)

	users := bluedb.QueryUsers(params)
	logs.Debug("users:%v", users)
	if len(users) <= 0 {
		users = []bluedb.User{}
	}
	var u = User{}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u.dbListObjectTrans(users))
}

type CreateUserResponse struct {
	ProjectId string `json:"project_id"`
}

type UserLoginResponse struct {
	ProjectId string `json:"project_id"`
	Token     string `json:"token"`
}

func UserLogin(w http.ResponseWriter, req *http.Request, _ map[string]string) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var userReq = &User{}
	err = json.Unmarshal(body, userReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	var user *bluedb.User
	if len(userReq.Name) > 0 {
		user = bluedb.QueryUserByName(userReq.Name)
		if user == nil || user.Passwd != userReq.Passwd {
			strErr := "invalid user or passwd."
			logs.Error(strErr)
			DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
			return
		}
	} else if len(userReq.Email) > 0 {
		user = bluedb.QueryUserByEmail(userReq.Email)
		if user == nil || user.Passwd != userReq.Passwd {
			strErr := "invalid email or passwd."
			logs.Error(strErr)
			DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
			return
		}
	} else {
		strErr := "invalid user or passwd."
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UserLoginResponse{
		ProjectId: user.Id,
		Token:     utils.GenToken(user.Id, userReq.Passwd),
	})
	w.WriteHeader(http.StatusOK)
}

func CreateUser(w http.ResponseWriter, req *http.Request, _ map[string]string) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var userReq = &User{}
	err = json.Unmarshal(body, userReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(userReq.Name)
	if len(name) <= 0 || len(name) >= 60 {
		strErr := fmt.Sprintf("Name(%s) is empty or exceed 60 bytes.", userReq.Name)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}
	passwd := strings.TrimSpace(userReq.Passwd)
	if len(passwd) <= 6 || len(passwd) > 120 {
		strErr := fmt.Sprintf("Passwd is less 6 or exceed 120 bytes.")
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(userReq.Email)
	if len(email) <= 0 || !strings.Contains(email, "@") {
		strErr := fmt.Sprintf("Email is invalid.")
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	user := bluedb.QueryUserByEmail(email)
	if user != nil {
		strErr := fmt.Sprintf("Email(%s) has been registed.", email)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	user = bluedb.QueryUserByName(name)
	if user != nil {
		strErr := fmt.Sprintf("username(%s) has been registed.", name)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	logs.Info("create user:%v", userReq)
	var userDb = bluedb.User{
		Name:    name,
		Passwd:  passwd,
		Email:   email,
		Mobile:  userReq.Mobile,
		Address: userReq.Address,
	}
	id := bluedb.CreateUser(userDb)
	if len(id) <= 0 {
		logs.Error("Create user fail. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	//notify mqtt
	if mqttclient.Client != nil {
		mqttclient.Client.NotifyUserAdd(name, passwd, id)
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateUserResponse{ProjectId: id})
}

func BindAwsUser(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	user, err := bluedb.QueryUserById(projectId)
	if err != nil {
		logs.Debug("get user err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	if len(user.AwsUsername) > 0 {
		errStr := fmt.Sprintf("user(%s) has binded aws-user(%s)", user.Name, user.AwsUsername)
		logs.Error(errStr)
		DefaultHandler.ServeHTTP(w, req, errors.New(errStr), http.StatusBadRequest)
		return
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var bindReq = &BindAwsUserReq{}
	err = json.Unmarshal(body, bindReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(bindReq.Name)
	if len(name) <= 0 {
		strErr := fmt.Sprintf("name is empty.")
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}
	if err := aws.CheckAkSk(bindReq.AccessKey, bindReq.SecretKey); err != nil {
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	user.AwsUsername = bindReq.Name
	user.AccessKey = bindReq.AccessKey
	user.SecretKey = bindReq.SecretKey
	err = bluedb.UpdateUser(user)
	if err != nil {
		logs.Error("update user fail. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteUser(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	id := ps["projectId"]
	user, _ := bluedb.QueryUserById(id)
	err := bluedb.DeleteUser(id)
	if err != nil {
		logs.Error("Delete user failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	//notify mqtt
	if len(user.Name) > 0 && mqttclient.Client != nil {
		mqttclient.Client.NotifyUserDelete(user.Name)
	}
	w.WriteHeader(http.StatusOK)

}

func GetUser(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	id := ps["projectId"]
	user, err := bluedb.QueryUserById(id)
	if err != nil {
		logs.Error("Get user fail. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	logs.Debug("user:%v", user)
	var u = User{}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u.dbObjectTrans(user))
}
