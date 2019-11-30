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

type UserNotifyReq struct {
	Email  string `json:"email"`
	Mobile string `json:"mobile"`
}

type UserNotify struct {
	Emails  []string `json:"emails"`
	Mobiles []string `json:"mobiles"`
}

func topicName(projectId string) string {
	return strings.Replace(projectId, "-", "", -1)
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
	svc := sns.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})

	// check topic
	input := sns.GetEndpointAttributesInput{
		EndpointArn: &tpc,
	}
	_, err = svc.GetEndpointAttributes(&input)
	if err != nil {
		if strings.Contains(err.Error(), sns.ErrCodeResourceNotFoundException) {
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
			userNotifies.Mobiles = append(userNotifies.Mobiles, *sub.Endpoint)
		} else if *sub.Protocol == "email" {
			userNotifies.Emails = append(userNotifies.Emails, *sub.Endpoint)
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
	if len(addReq.Email) > 0{
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
	var rmvReq UserNotifyReq
	err = json.Unmarshal(body, &rmvReq)
	if err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	logs.Info("remove notify:%v", rmvReq)
	protocal := ""
	endpoint := ""
	if len(rmvReq.Email) > 0{
		protocal = "email"
		endpoint = rmvReq.Email
	} else if len(rmvReq.Mobile) > 0 {
		protocal = "sms"
		endpoint = rmvReq.Mobile
	}

	// found and delete it
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
	var subTarget *sns.Subscription
	for _, sub := range subs.Subscriptions {
		if *sub.Protocol == protocal && *sub.Endpoint == endpoint {
			subTarget = sub
		}
	}
	if subTarget == nil {
		logs.Error("not found %s", endpoint)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(fmt.Sprintf("not found %s", endpoint)))
		return
	}
	subDel := sns.UnsubscribeInput{
		SubscriptionArn: subTarget.SubscriptionArn,
	}
	_, err = svc.Unsubscribe(&subDel)
	if err != nil {
		logs.Error("remove notify fail. err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	logs.Info("remove subscribe(%s) success", endpoint)
}