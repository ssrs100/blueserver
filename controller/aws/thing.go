package aws

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/influxdb"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	region = "us-west-2"

	owner = "owner"
)

type RegisterThingReq struct {
	Name        string `json:"name"`
	WifiAddr    string `json:"wifi_addr"`
	EtherAddr   string `json:"ether_addr"`
	Description string `json:"description"`
}

type UpdateThingReq struct {
	Description string `json:"description"`
}

type Thing struct {
	Id          string     `json:"id"`
	Name        string     `json:"name"`
	AwsName     string     `json:"aws_name"`
	AwsArn      string     `json:"aws_arn"`
	ProjectId   string     `json:"project_id"`
	Status      int        `json:"status"`
	WifiAddr    string     `json:"wifi_addr"`
	EtherAddr   string     `json:"ether_addr"`
	Description string     `json:"description"`
	CreateAt    *time.Time `json:"create_at"`
}

type ThingsWrap struct {
	Things []*Thing `json:"things"`
}

func awsTingName(name, projectId string) string {
	return name
}

func checkProject(projectId string) (*bluedb.User, error) {
	u, err := bluedb.QueryUserById(projectId)
	if err != nil {
		logs.Error("err:%s", err.Error())
		return nil, errors.New("project id not found")
	}
	if len(u.AccessKey) == 0 || len(u.SecretKey) == 0 {
		logs.Info("%s ak/sk is empty, ready to create", u.Name)
		u, err = bindAwsUser(u)
		if err != nil {
			logs.Error("err:%s", err.Error())
			return nil, errors.New("bind aws user fail")
		}
	}
	return &u, nil
}

func RegisterThing(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	u, err := checkProject(projectId)
	if err != nil {
		logs.Error("checkProject err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer req.Body.Close()

	var register = RegisterThingReq{}
	if err = json.Unmarshal(body, &register); err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	// check thing name
	if len(register.Name) == 0 || strings.Contains(register.Name, ":") {
		errStr := fmt.Sprintf("invalid name:%s.", register.Name)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(errStr))
		return
	}
	existThing := bluedb.GetThingByName(register.Name)
	if existThing != nil {
		errStr := fmt.Sprintf("thing(%s) has been used.", register.Name)
		logs.Error("%s has been used.", register.Name)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(errStr))
		return
	}

	// create thing
	logs.Info("create thing with ak(%s) sk(%s)", u.AccessKey, u.SecretKey)
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		u.AccessKey,
		u.SecretKey,
		"",
	)

	svc := iot.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})

	attr := make(map[string]*string)
	attr["owner"] = &u.Name
	attr["wifi_addr"] = &register.WifiAddr
	attr["ether_addr"] = &register.EtherAddr
	//attr["description"] = &register.Description
	attrThing := iot.AttributePayload{
		Attributes: attr,
	}

	awsThingName := awsTingName(register.Name, projectId)
	awsReq := iot.CreateThingInput{
		ThingName:        &awsThingName,
		AttributePayload: &attrThing,
	}
	thingOut, err := svc.CreateThing(&awsReq)
	if err != nil {
		logs.Error("create thing err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	certUrn := bluedb.GetSys("certUrn")
	attachReq := iot.AttachThingPrincipalInput{
		ThingName: &awsThingName,
		Principal: &certUrn,
	}
	_, err = svc.AttachThingPrincipal(&attachReq)
	if err != nil {
		logs.Error("attach principal err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	t := bluedb.Thing{
		Name:        register.Name,
		AwsName:     awsThingName,
		AwsArn:      *thingOut.ThingArn,
		ProjectId:   projectId,
		WifiAddr:    register.WifiAddr,
		EtherAddr:   register.EtherAddr,
		Description: register.Description,
	}
	if err := bluedb.SaveThing(t); err != nil {
		logs.Error("save thing err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func bindAwsUser(user bluedb.User) (bluedb.User, error) {
	admin, err := bluedb.QueryUserByName("admin")
	if err != nil {
		logs.Error("get admin err:%s", err.Error())
		return user, errors.New("get admin err")
	}
	if admin == nil {
		logs.Error("admin user not found")
		return user, errors.New("admin user not found")
	}
	//sess := session.Must(session.NewSession())
	//creds := credentials.NewStaticCredentials(
	//	admin.AccessKey,
	//	admin.SecretKey,
	//	"",
	//)
	//
	//awsUserName := user.Name
	//
	//svc := iam.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})
	//createReq := iam.CreateUserInput{
	//	UserName: &awsUserName,
	//}
	//logs.Info("create user")
	//_, err := svc.CreateUser(&createReq)
	//if err != nil {
	//	logs.Error("create aws user fail, err:%s", err.Error())
	//	return user, err
	//}
	//
	//logs.Info("bind policy")
	//iamPolicy := bluedb.GetSys("iamPolicy")
	//addPolicyReq := iam.AttachUserPolicyInput{
	//	UserName:  &awsUserName,
	//	PolicyArn: &iamPolicy,
	//}
	//_, err = svc.AttachUserPolicy(&addPolicyReq)
	//if err != nil {
	//	logs.Error("add user(%s) to policy fail, err:%s", user.Name, err.Error())
	//	return user, err
	//}
	//
	//logs.Info("bind group")
	//iamGroup := bluedb.GetSys("iamGroup")
	//addGroupReq := iam.AddUserToGroupInput{
	//	UserName:  &awsUserName,
	//	GroupName: &iamGroup,
	//}
	//_, err = svc.AddUserToGroup(&addGroupReq)
	//if err != nil {
	//	logs.Error("add user(%s) to group fail, err:%s", user.Name, err.Error())
	//	return user, err
	//}
	//
	//logs.Info("create ak sk")
	//akReq := iam.CreateAccessKeyInput{
	//	UserName: &awsUserName,
	//}
	//akOut, err := svc.CreateAccessKey(&akReq)
	//if err != nil {
	//	logs.Error("create ak(%s) fail, err:%s", user.Name, err.Error())
	//	return user, err
	//}
	//if akOut == nil ||
	//	akOut.AccessKey == nil ||
	//	akOut.AccessKey.AccessKeyId == nil ||
	//	akOut.AccessKey.SecretAccessKey == nil {
	//	logs.Error("create ak(%s) fail, no invalid ak sk", user.Name)
	//	return user, errors.New("no invalid ak sk")
	//}
	//ak := akOut.AccessKey.AccessKeyId
	//sk := akOut.AccessKey.SecretAccessKey
	//logs.Info("user(%s) ak(%s) sk(%s) saved", awsUserName, *ak, *sk)
	//user.AwsUsername = awsUserName
	//user.AccessKey = *ak
	//user.SecretKey = *sk
	user.AwsUsername = admin.AwsUsername
	user.AccessKey = admin.AccessKey
	user.SecretKey = admin.SecretKey
	if err := bluedb.UpdateUser(user); err != nil {
		logs.Error("update user(%s) fail, err:%s", user.Name, err.Error())
		return user, err
	}
	return user, nil
}

func UpdateThing(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	thingName := ps["thingName"]
	//u, err := checkProject(projectId)
	//if err != nil {
	//	logs.Error("checkProject err:%s", err.Error())
	//	w.WriteHeader(http.StatusBadRequest)
	//	_, _ = w.Write([]byte(err.Error()))
	//	return
	//}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer req.Body.Close()

	var update = UpdateThingReq{}
	if err = json.Unmarshal(body, &update); err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	existThing := bluedb.GetThing(projectId, thingName)
	if existThing == nil {
		errStr := fmt.Sprintf("%s not exist.", thingName)
		logs.Error("%s not exist.", thingName)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(errStr))
		return
	}

	// update thing
	logs.Info("update thing..")
	//sess := session.Must(session.NewSession())
	//creds := credentials.NewStaticCredentials(
	//	u.AccessKey,
	//	u.SecretKey,
	//	"",
	//)
	//
	//svc := iot.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})
	//
	//attr := make(map[string]*string)
	//attr["description"] = &update.Description
	//attrThing := iot.AttributePayload{
	//	Attributes: attr,
	//}
	//
	//awsThingName := existThing.AwsName
	//awsReq := iot.UpdateThingInput{
	//	ThingName:        &awsThingName,
	//	AttributePayload: &attrThing,
	//}
	//_, err = svc.UpdateThing(&awsReq)
	//if err != nil {
	//	logs.Error("update thing err:%s", err.Error())
	//	w.WriteHeader(http.StatusInternalServerError)
	//	_, _ = w.Write([]byte(err.Error()))
	//	return
	//}

	existThing.Description = update.Description
	if err := bluedb.UpdateThing(*existThing); err != nil {
		logs.Error("update db thing err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func RemoveThing(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	thingName := ps["thingName"]
	u, err := bluedb.QueryUserById(projectId)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("project id not found"))
		return
	}

	existThing := bluedb.GetThing(projectId, thingName)
	if existThing == nil {
		errStr := fmt.Sprintf("%s is not exist.", thingName)
		logs.Error(errStr)
		w.WriteHeader(http.StatusOK)
		return
	}

	if len(u.AccessKey) == 0 || len(u.SecretKey) == 0 {
		logs.Info("%s ak/sk is empty", u.Name)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("ak/sk is empty"))
		return
	}

	// create thing
	logs.Info("remove thing(%s)", thingName)
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		u.AccessKey,
		u.SecretKey,
		"",
	)

	awsThingName := existThing.AwsName

	svc := iot.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})
	certUrn := bluedb.GetSys("certUrn")
	detachReq := iot.DetachThingPrincipalInput{
		ThingName: &awsThingName,
		Principal: &certUrn,
	}
	_, err = svc.DetachThingPrincipal(&detachReq)
	if err != nil && !isNotFound(err) {
		logs.Error("detach principal err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	awsReq := iot.DeleteThingInput{
		ThingName: &awsThingName,
	}
	_, err = svc.DeleteThing(&awsReq)
	if err != nil && !isNotFound(err) {
		logs.Error("remove thing err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	if err := bluedb.DeleteThing(existThing.Id); err != nil {
		logs.Error("remove db thing err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	if err := influxdb.DeleteData(influxdb.TableTemperature, thingName, projectId); err != nil {
		logs.Error(err.Error())
	}
	if err := influxdb.DeleteData(influxdb.TableBroadcast, thingName, projectId); err != nil {
		logs.Error(err.Error())
	}

	w.WriteHeader(http.StatusOK)
}

func isNotFound(e error) bool {
	if strings.Contains(e.Error(), iot.ErrCodeResourceNotFoundException) {
		return true
	}
	return false
}

func GetThingLatestData(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	thingName := ps["thingName"]
	projectId := ps["projectId"]
	existThing := bluedb.GetThing(projectId, thingName)
	if existThing == nil {
		logs.Error("not found thing %s", thingName)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("thing name not found"))
		return
	}
	device := req.URL.Query().Get("device")
	data, err := influxdb.GetLatest(getDataType(req), thingName, device, projectId)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("thing id not found"))
		return
	}
	body, err := json.Marshal(data)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(body)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func GetThingData(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	thingName := ps["thingName"]
	projectId := ps["projectId"]
	existThing := bluedb.GetThing(projectId, thingName)
	if existThing == nil {
		logs.Error("not found thing %s", thingName)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("thing name not found"))
		return
	}
	startAt := req.URL.Query().Get("startAt")
	endAt := req.URL.Query().Get("endAt")
	device := req.URL.Query().Get("device")
	// startAt, endAt like '2019-08-17T06:40:27.995Z'
	var tEnd time.Time
	tStart, err := time.Parse(time.RFC3339, startAt)
	if err == nil {
		tEnd, err = time.Parse(time.RFC3339, endAt)
	}
	if err != nil || tEnd.Before(tStart) {
		strErr := fmt.Sprintf("Invalid time params, startAt:%s, endAt:%s.", startAt, endAt)
		logs.Error(strErr)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strErr))
		return
	}
	datas, err := influxdb.GetDataByTime(getDataType(req), thingName, startAt, endAt, device, projectId)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("thing id not found"))
		return
	}
	list := influxdb.OutDataList{
		Datas: datas,
		Count: len(datas),
	}
	body, err := json.Marshal(list)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(body)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func GetThingDevice(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	thingName := ps["thingName"]
	projectId := ps["projectId"]
	existThing := bluedb.GetThing(projectId, thingName)
	if existThing == nil {
		logs.Error("not found thing %s", thingName)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("thing name not found"))
		return
	}
	devices, err := influxdb.GetDevicesByThing(getDataType(req), thingName, projectId)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("thing id not found"))
		return
	}
	list := influxdb.DeviceList{
		Devices: devices,
	}
	body, err := json.Marshal(list)
	if err != nil {
		logs.Error("Invalid data. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(body)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func ListThingsV2(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	params := make(map[string]interface{})
	params["project_id"] = projectId
	limit := req.URL.Query().Get("limit")
	offset := req.URL.Query().Get("offset")
	if l, err := strconv.Atoi(limit); err == nil {
		params["limit"] = l
	}

	if o, err := strconv.Atoi(offset); err == nil {
		params["offset"] = o
	}

	things := bluedb.QueryThings(params)
	ret := make([]*Thing, 0)
	for _, t := range things {
		o := Thing{
			Id:          t.Id,
			Name:        t.Name,
			AwsName:     t.AwsName,
			AwsArn:      t.AwsArn,
			ProjectId:   t.ProjectId,
			Status:      t.Status,
			WifiAddr:    t.WifiAddr,
			EtherAddr:   t.EtherAddr,
			Description: t.Description,
			CreateAt:    t.CreateAt,
		}
		ret = append(ret, &o)
	}
	list := ThingsWrap{
		Things: ret,
	}
	outBytes, err := json.Marshal(list)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errStr := fmt.Sprintf("build json err:%s", err.Error())
		_, _ = w.Write([]byte(errStr))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(outBytes)
}

func ListThings(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	u, err := bluedb.QueryUserById(projectId)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("project id not found"))
		return
	}

	if len(u.AccessKey) == 0 || len(u.SecretKey) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("user project id not bind aws"))
		return
	}

	limit := req.URL.Query().Get("maxResults")
	nextToken := req.URL.Query().Get("nextToken")
	thingTypeName := req.URL.Query().Get("thingTypeName")
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		u.AccessKey,
		u.SecretKey,
		"",
	)
	svc := iot.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})
	awsReq := iot.ListThingsInput{
		AttributeName:  &owner,
		AttributeValue: &u.Name,
		NextToken:      nil,
	}
	if len(limit) > 0 {
		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			logs.Error("limit is invalid")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("limit is invalid"))
			return
		}
		limit64 := int64(limitInt)
		awsReq.MaxResults = &limit64
	}

	if len(nextToken) > 0 {
		awsReq.NextToken = &nextToken
	}
	if len(thingTypeName) > 0 {
		awsReq.ThingTypeName = &thingTypeName
	}

	rsp, err := listThings(svc, &awsReq)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		errStr := fmt.Sprintf("aws return err:%s", err.Error())
		_, _ = w.Write([]byte(errStr))
		return
	}
	outBytes, err := jsonutil.BuildJSON(rsp)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errStr := fmt.Sprintf("aws build json err:%s, rsp:%s", err.Error(), rsp.String())
		_, _ = w.Write([]byte(errStr))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(outBytes)
}

func listThings(svc *iot.IoT, input *iot.ListThingsInput) (*iot.ListThingsOutput, error) {
	req, out := svc.ListThingsRequest(input)
	req.HTTPRequest.Header.Add("Accept", "application/json")
	return out, req.Send()
}

func CheckAkSk(ak, sk string) error {
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		ak,
		sk,
		"",
	)
	svc := iot.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})
	limit64 := int64(1)
	awsReq := iot.ListThingsInput{
		NextToken:  nil,
		MaxResults: &limit64,
	}

	_, err := listThings(svc, &awsReq)
	if err != nil {
		errStr := fmt.Sprintf("aws return err:%s", err.Error())
		logs.Error(errStr)
		return err
	}
	return nil
}
