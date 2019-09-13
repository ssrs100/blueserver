package awsmqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/jack0liu/utils"
	"github.com/patrickmn/go-cache"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/influxdb"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"time"
)

var _ aws.Config
var _ awserr.Error
var _ request.Request

type AwsIotClient struct {
	reportChan chan *Shadow
	awsClient  *Client
	snsClient  *sns.SNS
	user       *bluedb.User
}

var msgTemplate = "[notice]device(%s) thing(%s) temperature is %v, it has exceeded threshold, please pay attention to it."

var cleanTemplate = "[clean]device(%s) thing(%s) temperature is %v, it drops below threshold."

var deviceSnsCache *cache.Cache

var useClientCache map[string]*AwsIotClient

var stopChan chan interface{}

func init() {
	deviceSnsCache = cache.New(24*time.Hour, 30*time.Minute)
	useClientCache = make(map[string]*AwsIotClient)
}

func listAllDir(path string) []string {
	readerInfos, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err.Error())
	}
	dirs := make([]string, 0)
	for _, info := range readerInfos {
		if info.IsDir() {
			dirs = append(dirs, info.Name())
		}
	}
	return dirs
}

func InitAwsClient() {
	baseDir := utils.GetBasePath()
	certDir := filepath.Join(baseDir, "conf", "cert")
	users := listAllDir(certDir)
	for _, u := range users {
		user := bluedb.QueryUserByName(u)
		if user == nil || len(user.AccessKey) == 0 || len(user.SecretKey) == 0 {
			logs.Error("get user(%s) fail, or no ak sk set", u)
			continue
		}
		c := conf.LoadFile(filepath.Join(certDir, u, "conf.json"))
		if c == nil {
			logs.Error("load user(%s) conf.json fail", u)
			continue
		}
		iotEndpoint := conf.GetString("iot_endpoint")
		client, err := NewClient(
			KeyPair{
				PrivateKeyPath:    filepath.Join(certDir, u, "private.pem.key"),
				CertificatePath:   filepath.Join(certDir, u, "certificate.pem.crt"),
				CACertificatePath: filepath.Join(certDir, "AmazonRootCA1.pem"),
			},
			iotEndpoint, // AWS IoT endpoint
			u,
		)
		if err != nil {
			panic(err)
		}

		awsIC := AwsIotClient{}
		awsIC.awsClient = client
		awsIC.reportChan, err = awsIC.awsClient.SubscribeForThingReport()
		if err != nil {
			logs.Error("subscribe user(%s) thing report fail", u)
			continue
		}
		go awsIC.startAwsClient(user.Id, stopChan)
		useClientCache[u] = &awsIC
	}
	logs.Info("start aws client success")
	<-stopChan
}

func (ac *AwsIotClient) startAwsClient(projectId string, stop chan interface{}) {
	tempThresh := conf.GetIntWithDefault("temperature_thresh", 30)
	for {
		select {
		case s, ok := <-ac.reportChan:
			if !ok {
				logs.Debug("failed to read from shadow channel")
			} else {
				var rd influxdb.ReportData
				if err := json.Unmarshal(s.Msg, &rd); err != nil {
					logs.Error("err:%s", err.Error())
					continue
				}
				rd.Thing = s.Thing
				if err := influxdb.Insert("temperature", &rd); err != nil {
					logs.Error("%s", err.Error())
					continue
				}
				var tmp int
				tmpFloat, err := strconv.ParseFloat(string(rd.Temperature), 64)
				if err != nil {
					tmp, err = strconv.Atoi(string(rd.Temperature))
					if err != nil {
						logs.Error("temper err:%v", rd.Temperature)
					}
				} else {
					tmp = int(tmpFloat)
				}
				if tmp >= tempThresh {
					go ac.sendSns(&rd, false)
				} else {
					go ac.sendSns(&rd, true)
				}
				logs.Debug("insert %v", rd)
				logs.Debug("insert influxdb success")
			}
		case <-stop:
			logs.Info("stopped")
		}
	}
}

func (ac *AwsIotClient) initSns() {
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		ac.user.AccessKey,
		ac.user.SecretKey,
		"",
	)
	ac.snsClient = sns.New(sess, &aws.Config{Credentials: creds, Region: aws.String("us-west-2")})
}

func (ac *AwsIotClient) sendSns(data *influxdb.ReportData, isClean bool) {
	if isClean {
		ac.sendCleanMsg(data)
	} else {
		ac.sendNotifyMsg(data)
	}
}

func (ac *AwsIotClient) sendNotifyMsg(data *influxdb.ReportData) {
	_, ok := deviceSnsCache.Get(data.Device)
	if ok {
		return
	}
	d, _ := bluedb.QueryNoticeByDevice(data.Device)
	if d != nil {
		return
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	msg := fmt.Sprintf(msgTemplate, data.Device, data.Thing, data.Temperature)
	params := &sns.PublishInput{
		Message:  aws.String(msg),
		TopicArn: aws.String("arn:aws:sns:us-west-2:415890359503:email"),
	}
	_, err := ac.snsClient.PublishWithContext(ctx, params)
	if err != nil {
		logs.Error("publish err:%s", err.Error())
		aerr, ok := err.(awserr.RequestFailure)
		if !ok {
			logs.Error("expect awserr")
			return
		}
		logs.Error("expect awserr code:%v, msg:%s", aerr.Code(), aerr.Message())
		return
	}
	logs.Info("send(%s) notify to sns success", data.Device)
	n := bluedb.Notify{
		Device:  data.Device,
		Noticed: "1",
	}
	deviceSnsCache.Set(data.Device, "1", cache.DefaultExpiration)
	bluedb.SaveNotice(n)
}

func (ac *AwsIotClient) sendCleanMsg(data *influxdb.ReportData) {
	_, ok := deviceSnsCache.Get(data.Device)
	if !ok {
		d, _ := bluedb.QueryNoticeByDevice(data.Device)
		if d == nil {
			return
		}
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	msg := fmt.Sprintf(cleanTemplate, data.Device, data.Thing, data.Temperature)
	params := &sns.PublishInput{
		Message:  aws.String(msg),
		TopicArn: aws.String("arn:aws:sns:us-west-2:415890359503:email"),
	}
	_, err := ac.snsClient.PublishWithContext(ctx, params)
	if err != nil {
		logs.Error("publish err:%s", err.Error())
		aerr, ok := err.(awserr.RequestFailure)
		if !ok {
			logs.Error("expect awserr")
			return
		}
		logs.Error("expect awserr code:%v, msg:%s", aerr.Code(), aerr.Message())
		return
	}
	logs.Info("send(%s) clean to sns success", data.Device)
	deviceSnsCache.Delete(data.Device)
	bluedb.DeleteNotice(data.Device)
}
