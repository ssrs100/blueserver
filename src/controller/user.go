package controller

import (
	"bluedb"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"logs"
	"mqttclient"
	"net/http"
	"strconv"
	"strings"
	"utils"
)

type User struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Passwd  string `json:"passwd"`
	Email   string `json:"email"`
	Mobile  string `json:"mobile"`
	Address string `json:"address"`
}

var (
	log = logs.GetLogger()
)

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

func GetUsers(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
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

	log.Debug("params:%v", params)

	users := bluedb.QueryUsers(params)
	log.Debug("users:%v", users)
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

func UserLogin(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var userReq = &User{}
	err = json.Unmarshal(body, userReq)
	if err != nil {
		log.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}

	var user *bluedb.User
	if len(userReq.Name) > 0 {
		user = bluedb.QueryUserByName(userReq.Name)
		if user == nil || user.Passwd != userReq.Passwd {
			strErr := "invalid user or passwd."
			log.Error(strErr)
			DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
			return
		}
	} else if len(userReq.Email) > 0 {
		user = bluedb.QueryUserByEmail(userReq.Email)
		if user == nil || user.Passwd != userReq.Passwd {
			strErr := "invalid email or passwd."
			log.Error(strErr)
			DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
			return
		}
	} else {
		strErr := "invalid user or passwd."
		log.Error(strErr)
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

func CreateUser(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var userReq = &User{}
	err = json.Unmarshal(body, userReq)
	if err != nil {
		log.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(userReq.Name)
	if len(name) <= 0 || len(name) >= 60 {
		strErr := fmt.Sprintf("Name(%s) is empty or exceed 60 bytes.", userReq.Name)
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
                return
	}
	passwd := strings.TrimSpace(userReq.Passwd)
	if len(passwd) <= 6 || len(passwd) > 120 {
		strErr := fmt.Sprintf("Passwd is less 6 or exceed 120 bytes.")
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
                return
	}

	email := strings.TrimSpace(userReq.Email)
	if len(email) <= 0 || !strings.Contains(email, "@") {
		strErr := fmt.Sprintf("Email is invalid.")
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
                return
	}

	user := bluedb.QueryUserByEmail(email)
	if user != nil {
		strErr := fmt.Sprintf("Email(%s) has been registed.", email)
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	user = bluedb.QueryUserByName(name)
	if user != nil {
		strErr := fmt.Sprintf("username(%s) has been registed.", name)
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	log.Info("create user:%v", userReq)
	var userDb = bluedb.User{
		Name:    name,
		Passwd:  passwd,
		Email:   email,
		Mobile:  userReq.Mobile,
		Address: userReq.Address,
	}
	id := bluedb.CreateUser(userDb)
	if len(id) <= 0 {
		log.Error("Create user fail. err:%s", err.Error())
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

func DeleteUser(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("projectId")
	user, _ := bluedb.QueryUserById(id)
	err := bluedb.DeleteUser(id)
	if err != nil {
		log.Error("Delete user failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	//notify mqtt
	if len(user.Name) > 0 && mqttclient.Client != nil {
		mqttclient.Client.NotifyUserDelete(user.Name)
	}
	w.WriteHeader(http.StatusOK)

}

func GetUser(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("projectId")
	user, err := bluedb.QueryUserById(id)
	if err != nil {
		log.Error("Get user fail. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	log.Debug("user:%v", user)
	var u = User{}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u.dbObjectTrans(user))
}
