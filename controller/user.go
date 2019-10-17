package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego/utils"
	"github.com/fernet/fernet-go"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/common"
	"github.com/ssrs100/blueserver/controller/aws"
	"github.com/ssrs100/blueserver/controller/middleware"
	"github.com/ssrs100/blueserver/mqttclient"
	"github.com/ssrs100/blueserver/sesscache"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	Confirmed   = 1
	UnConfirmed = 0
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
	logs.Info("login user start...")
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
		user, err = bluedb.QueryUserByName(userReq.Name)
		if err != nil {
			logs.Error("get user err:%s", err.Error())
			DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
			return
		}
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

	// gen token
	now := time.Now().UTC()
	us := middleware.UserSession{
		UserId:    user.Id,
		Roles:     []string{"te_admin"},
		CreatedAt: now.Format(time.RFC3339),
		ExpiredAt: now.Add(time.Hour * 24).Format(time.RFC3339),
	}
	key := fernet.Key{}
	err = key.Generate()
	if err != nil {
		logs.Error("Invalid key. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}
	sId := key.Encode()
	sess, err := json.Marshal(&us)
	if err != nil {
		logs.Error("Invalid sess. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}
	k := fernet.MustDecodeKeys(sId)
	tok, err := fernet.EncryptAndSign(sess, k[0])
	if err != nil {
		logs.Error("encrypt sess. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  common.CookieSessionId,
		Value: string(tok),
		Path:  "/",
	})
	sesscache.SetWithNoExpired("lastLogin_"+us.UserId, time.Now().Format(time.RFC3339))
	sesscache.Set(string(tok), sId)
	logs.Info("key:%s", sId)
	logs.Info("session:%s", string(tok))
	// return
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UserLoginResponse{
		ProjectId: user.Id,
		Token:     string(tok),
	})
	w.WriteHeader(http.StatusOK)
}

func ActiveUser(w http.ResponseWriter, req *http.Request, _ map[string]string) {
	token := req.URL.Query().Get("token")
	k := sesscache.Get(token)
	if len(k) == 0 {
		logs.Error("invalid active token")
		redirectAddr := conf.GetString("redirect_addr")
		http.Redirect(w, req, redirectAddr, http.StatusFound)
		return
	}
	logs.Info("key:%s", k)
	keys := fernet.MustDecodeKeys(k)
	tokenStr := fernet.VerifyAndDecrypt([]byte(token), 0, keys)
	if len(tokenStr) == 0 {
		sesscache.Del(token)
		http.Error(w, http.StatusText(401), http.StatusUnauthorized)
		return
	}
	userId := string(tokenStr)
	u, err := bluedb.QueryUserById(userId)
	if err != nil {
		logs.Error("get user(%s) err:%s", userId, err.Error())
		http.Error(w, http.StatusText(401), http.StatusUnauthorized)
		return
	}
	u.Status = Confirmed
	if err := bluedb.UpdateUser(u); err != nil {
		logs.Error("update user(%s) err:%s", userId, err.Error())
		http.Error(w, http.StatusText(401), http.StatusUnauthorized)
		return
	}
	logs.Info("user(%s) active success", u.Email)
	redirectAddr := conf.GetString("redirect_addr")
	http.Redirect(w, req, redirectAddr, http.StatusFound)
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

	user, err = bluedb.QueryUserByName(name)
	if err != nil {
		strErr := fmt.Sprintf("get username(%s) err:%s.", name, err.Error())
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	userId := ""
	if user != nil && user.Status == Confirmed {
		strErr := fmt.Sprintf("username(%s) has been registed.", name)
		logs.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	} else if user == nil {
		logs.Info("create user:%v", userReq)
		var userDb = bluedb.User{
			Name:    name,
			Passwd:  passwd,
			Email:   email,
			Mobile:  userReq.Mobile,
			Address: userReq.Address,
			Status:  UnConfirmed,
		}
		userId = bluedb.CreateUser(userDb)
		if len(userId) <= 0 {
			logs.Error("Create user fail. err:%s", err.Error())
			DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
			return
		}
	} else {
		userId = user.Id
	}

	// gen token
	key := fernet.Key{}
	err = key.Generate()
	if err != nil {
		logs.Error("Invalid key. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}
	sId := key.Encode()
	sess, err := json.Marshal(&userId)
	if err != nil {
		logs.Error("Invalid sess. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}
	k := fernet.MustDecodeKeys(sId)
	tok, err := fernet.EncryptAndSign(sess, k[0])
	if err != nil {
		logs.Error("encrypt sess. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusInternalServerError)
		return
	}

	if err := sendActivateEmail([]string{email}, string(tok)); err != nil {
		logs.Error("send email err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	sesscache.SetWithExpired(string(tok), sId, 20*time.Minute)

	//notify mqtt
	//if mqttclient.Client != nil {
	//	mqttclient.Client.NotifyUserAdd(name, passwd, userId)
	//}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateUserResponse{ProjectId: userId})
}

func sendActivateEmail(toUserEmails []string, token string) error {
	email := bluedb.GetSys("sysEmailUser")
	pwd := bluedb.GetSys("sysEmailPwd")
	config := fmt.Sprintf(`{"username":"%s","password":"%s","host":"smtp.exmail.qq.com","port":25}`, email, pwd)
	temail := utils.NewEMail(config)
	temail.To = toUserEmails
	temail.From = email
	temail.Subject = "Please verify your email for your Feasycom Account"

	redirectAddr := conf.GetString("redirect_addr")
	temail.HTML = "Please verify your email address by clicking the following link:<br/>" +
		"<href>" + redirectAddr + "/active?token=" + token + "</href><br/>It will expire in 20 minutes."

	err := temail.Send()
	if err != nil {
		logs.Error("send email fail, err:%s", err.Error())
		return err
	}
	return nil
}

func BindAwsUser(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	logs.Info("bind user start...")
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
