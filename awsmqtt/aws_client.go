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
	"path/filepath"
	"time"
)

var _ aws.Config
var _ awserr.Error
var _ request.Request

var (
	reportChan chan *Shadow
	awsClient  *Client
	snsClient  *sns.SNS
)

var msgTemplate = "[notice]device(%s) thing(%s) temperature is %d, it has exceeded threshold, please pay attention to it."

var cleanTemplate = "[clean]device(%s) thing(%s) temperature is %d, it drops below threshold."

var deviceSnsCache *cache.Cache

func init() {
	deviceSnsCache = cache.New(24*time.Hour, 30*time.Minute)
}

func InitAwsClient() {
	baseDir := utils.GetBasePath()
	client, err := NewClient(
		KeyPair{
			PrivateKeyPath:    filepath.Join(baseDir, "conf", "private.pem.key"),
			CertificatePath:   filepath.Join(baseDir, "conf", "certificate.pem.crt"),
			CACertificatePath: filepath.Join(baseDir, "conf", "AmazonRootCA1.pem"),
		},
		conf.GetStringWithDefault("iot_endpoint", "a359ikotxsoxw8-ats.iot.us-west-2.amazonaws.com"), // AWS IoT endpoint
		"blueserverclient",
	)
	if err != nil {
		panic(err)
	}
	awsClient = client

	reportChan, err = awsClient.SubscribeForThingReport()
	if err != nil {
		panic(err)
	}

	startAwsClient()
}

func startAwsClient() {
	tempThresh := conf.GetIntWithDefault("temperature_thresh", 30)
	for {
		select {
		case s, ok := <-reportChan:
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
				if int(rd.Temperature) >= tempThresh {
					go sendSns(&rd, false)
				} else {
					go sendSns(&rd, true)
				}
				logs.Debug("insert %v", rd)
				logs.Debug("insert influxdb success")
			}
		}
	}
}

func InitSns() {
	sess := session.Must(session.NewSession())
	adm := bluedb.QueryUserByName("admin")
	if adm == nil {
		panic("get admin fail")
	}
	creds := credentials.NewStaticCredentials(
		adm.AccessKey,
		adm.SecretKey,
		"blueSns",
	)
	snsClient = sns.New(sess, &aws.Config{Credentials: creds, Region: aws.String("us-west-2")})
}

func sendSns(data *influxdb.ReportData, isClean bool) {
	if isClean {
		sendCleanMsg(data)
	} else {
		sendNotifyMsg(data)
	}
}

func sendNotifyMsg(data *influxdb.ReportData) {
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
	_, err := snsClient.PublishWithContext(ctx, params)
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

func sendCleanMsg(data *influxdb.ReportData) {
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
	_, err := snsClient.PublishWithContext(ctx, params)
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
