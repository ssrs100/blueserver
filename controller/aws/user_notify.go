package aws

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/jack0liu/logs"
	"io/ioutil"
	"net/http"
	"strings"
)

type UserNotifyRmvReq struct {
	SubscribeId string `json:"subscribe_id"`
}

type UserNotifyReq struct {
	Email  string `json:"email"`
	Mobile string `json:"mobile"`
}

type NotifyInfo struct {
	Endpoint    string `json:"endpoint"`
	SubscribeId string `json:"subscribe_id"`
}

type UserNotify struct {
	Emails  []NotifyInfo `json:"emails"`
	Mobiles []NotifyInfo `json:"mobiles"`
}

func topicName(projectId string) string {
	//return strings.Replace(projectId, "-", "", -1)
	return projectId
}

func createTopic(svc *sns.SNS, projectId string) error {
	name := topicName(projectId)
	logs.Info("create topic name(%s)", name)
	displayName := "Temperature and humidity threshold notification"
	attr := make(map[string]*string)
	attr["DisplayName"] = &displayName
	crtTpc := &sns.CreateTopicInput{
		Attributes: attr,
		Name:       &name,
	}
	if _, err := svc.CreateTopic(crtTpc); err != nil {
		logs.Error("create topic err:%s", err.Error())
		return err
	}
	return nil
}

func GetUserNotify(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	u, err := checkProject(projectId)
	if err != nil {
		logs.Error("check project err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		u.AccessKey,
		u.SecretKey,
		"",
	)

	name := topicName(projectId)
	tpc := fmt.Sprintf("arn:aws:sns:us-west-2:415890359503:%s", name)
	logs.Debug(tpc)
	svc := sns.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})

	// check topic
	input := sns.GetTopicAttributesInput{
		TopicArn: &tpc,
	}
	_, err = svc.GetTopicAttributes(&input)
	if err != nil {
		if strings.Contains(err.Error(), sns.ErrCodeNotFoundException) {
			if err := createTopic(svc, projectId); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}
		} else {
			logs.Error("get topic err:%s", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	}
	// get subscribe
	listSub := sns.ListSubscriptionsByTopicInput{
		TopicArn: &tpc,
	}
	subs, err := svc.ListSubscriptionsByTopic(&listSub)
	if err != nil {
		logs.Error("list subscription err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	userNotifies := UserNotify{}
	for _, sub := range subs.Subscriptions {
		if *sub.Protocol == "sms" {
			e := NotifyInfo{
				Endpoint:    *sub.Endpoint,
				SubscribeId: *sub.SubscriptionArn,
			}
			userNotifies.Mobiles = append(userNotifies.Mobiles, e)
		} else if *sub.Protocol == "email" {
			e := NotifyInfo{
				Endpoint:    *sub.Endpoint,
				SubscribeId: *sub.SubscriptionArn,
			}
			userNotifies.Emails = append(userNotifies.Emails, e)
		}
	}
	ret, err := json.Marshal(&userNotifies)
	if err != nil {
		logs.Error("unmarshal err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(ret)
}

func AddUserNotify(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	u, err := checkProject(projectId)
	if err != nil {
		logs.Error("check project err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		u.AccessKey,
		u.SecretKey,
		"",
	)

	name := topicName(projectId)
	tpc := fmt.Sprintf("arn:aws:sns:us-west-2:415890359503:%s", name)
	svc := sns.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer req.Body.Close()
	var addReq UserNotifyReq
	err = json.Unmarshal(body, &addReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	logs.Info("add notify:%v", addReq)
	protocal := ""
	endpoint := ""
	if len(addReq.Email) > 0 {
		protocal = "email"
		endpoint = addReq.Email
	} else if len(addReq.Mobile) > 0 {
		protocal = "sms"
		endpoint = addReq.Mobile
	}
	subIn := sns.SubscribeInput{
		TopicArn: &tpc,
		Protocol: &protocal,
		Endpoint: &endpoint,
	}
	_, err = svc.Subscribe(&subIn)
	if err != nil {
		logs.Error("add notify fail. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("please confirm the subscribe(%s)", endpoint)))
}

func RmvUserNotify(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	u, err := checkProject(projectId)
	if err != nil {
		logs.Error("check project err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		u.AccessKey,
		u.SecretKey,
		"",
	)
	svc := sns.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer req.Body.Close()
	var rmvReq UserNotifyRmvReq
	err = json.Unmarshal(body, &rmvReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	logs.Info("remove notify:%v", rmvReq)

	// found and delete it
	subDel := sns.UnsubscribeInput{
		SubscriptionArn: &rmvReq.SubscribeId,
	}
	_, err = svc.Unsubscribe(&subDel)
	if err != nil {
		logs.Error("remove notify fail. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	logs.Info("remove subscribe(%s) success", rmvReq.SubscribeId)
}
