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
	"github.com/ssrs100/blueserver/influxdb"
	"path/filepath"
	"time"
)

var _ aws.Config
var _ awserr.Error
var _ request.Request

var (
	reportChan chan Shadow
	shadowChan chan Shadow
	awsClient  *Client
)

var msgTemplate = "device(%s) temperature is %d, it has exceeded threshold, please pay attention to it."

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
				if err := json.Unmarshal(s, &rd); err != nil {
					logs.Error("err:%s", err.Error())
					continue
				}
				if err := influxdb.Insert("temperature", &rd); err != nil {
					logs.Error("%s", err.Error())
					continue
				}
				if rd.Temperature > tempThresh {
					go sendSns(&rd)
				}
				logs.Debug("insert influxdb success")
			}
		}
	}
}

func sendSns(data *influxdb.ReportData) {
	sess := session.Must(session.NewSession())

	creds := credentials.NewStaticCredentials(
		"AKIAW7NVRWDYGTKQEM6G",
		"BdfR8KliCkW+p2IFBwC8zlm02bOXColzYgr4zpYS",
		"",
	)
	svc := sns.New(sess, &aws.Config{Credentials: creds, Region: aws.String("us-west-2")})
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	msg := fmt.Sprintf(msgTemplate, data.Device, data.Temperature)
	params := &sns.PublishInput{
		Message:  aws.String(msg),
		TopicArn: aws.String("arn:aws:sns:us-west-2:415890359503:email"),
	}
	_, err := svc.PublishWithContext(ctx, params)
	if err != nil {
		logs.Error("publish err:%s", err.Error())
		aerr, ok := err.(awserr.RequestFailure)
		if !ok {
			logs.Error("expect awserr")
			return
		}
		logs.Error("expect awserr code:%v, msg:%s", aerr.Code(), aerr.Message())
	}

}
