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
	"github.com/ssrs100/blueserver/common"
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

type thresh struct {
	minTemp int
	maxTemp int
	minHum  int
	maxHum  int
}

const (
	tempKey     = "temperature"
	humidityKey = "humidity"
)

var msgTemplate = "[notice]device(%s) thing(%s) %s is %v, it's out of the range of device settings, please pay attention to it."

var cleanTemplate = "[clean]device(%s) thing(%s) %s is %v, it restores back to the range of device settings."

var (
	deviceSnsCache *cache.Cache
	useClientCache map[string]*AwsIotClient
)

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
	tempMinThresh := conf.GetIntWithDefault("temperature_min_thresh", common.MinTemp)
	tempMaxThresh := conf.GetIntWithDefault("temperature_max_thresh", common.MaxTemp)
	humiMinThresh := conf.GetIntWithDefault("humi_min_thresh", common.MinHumi)
	humiMaxThresh := conf.GetIntWithDefault("humi_max_thresh", common.MaxHumi)
	defaultThresh := thresh{
		minTemp: tempMinThresh,
		maxTemp: tempMaxThresh,
		minHum:  humiMinThresh,
		maxHum:  humiMaxThresh,
	}
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
				rd.ProjectId = projectId
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

				// humidity
				var hum int
				humFloat, err := strconv.ParseFloat(string(rd.Humidity), 64)
				if err != nil {
					hum, err = strconv.Atoi(string(rd.Humidity))
					if err != nil {
						logs.Error("humidity err:%v", rd.Humidity)
					}
				} else {
					hum = int(humFloat)
				}

				threshDevice := getThresh(&rd, &defaultThresh)
				if tmp >= threshDevice.maxTemp {
					go ac.sendSns(tempKey, &rd, true, false)
				} else if tmp < threshDevice.minTemp {
					go ac.sendSns(tempKey, &rd, false, false)
				} else {
					go ac.sendSns(tempKey, &rd, false, true)
				}

				// humidity
				if hum >= threshDevice.maxHum {
					go ac.sendSns(humidityKey, &rd, true, false)
				} else if tmp < threshDevice.minHum {
					go ac.sendSns(humidityKey, &rd, false, false)
				} else {
					go ac.sendSns(humidityKey, &rd, false, true)
				}
				logs.Debug("insert %v", rd)
				logs.Debug("insert influxdb success")
			}
		case <-stop:
			logs.Info("stopped")
		}
	}
}

func getThresh(data *influxdb.ReportData, defaultThresh *thresh) *thresh {

	dt, _ := bluedb.QueryDevThresh(data.ProjectId, data.Device)
	if dt == nil {
		return defaultThresh
	}
	thre := thresh{
		minTemp: dt.TemperatureMin,
		maxTemp: dt.TemperatureMax,
		minHum:  dt.HumidityMin,
		maxHum:  dt.HumidityMax,
	}
	return &thre
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

func (ac *AwsIotClient) sendSns(key string, data *influxdb.ReportData, upperLimit, isClean bool) {
	if isClean {
		ac.sendCleanMsg(key, data)
	} else {
		ac.sendNotifyMsg(key, data, upperLimit)
	}
}

func (ac *AwsIotClient) sendNotifyMsg(key string, data *influxdb.ReportData, upperLimit bool) {
	dbKey := key + "upper"
	if !upperLimit {
		dbKey = key + "lower"
	}
	_, ok := deviceSnsCache.Get(data.Device + dbKey)
	if ok {
		return
	}
	d, _ := bluedb.QueryNoticeByDevice(data.Device, dbKey)
	if d != nil {
		return
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	value := data.Temperature
	switch key {
	case tempKey:
		value = data.Temperature
	case humidityKey:
		value = data.Humidity
	default:
		logs.Error("invalid key")
	}
	msg := fmt.Sprintf(msgTemplate, data.Device, data.Thing, key, value)
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
		Key:     dbKey,
	}
	deviceSnsCache.Set(data.Device+dbKey, "1", cache.DefaultExpiration)
	bluedb.SaveNotice(n)
}

func (ac *AwsIotClient) sendCleanMsg(key string, data *influxdb.ReportData) {
	_, ok := deviceSnsCache.Get(data.Device + key)
	if !ok {
		d, _ := bluedb.QueryNoticeByDevice(data.Device, key)
		if d == nil {
			return
		}
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	value := data.Temperature
	switch key {
	case tempKey:
		value = data.Temperature
	case humidityKey:
		value = data.Humidity
	default:
		logs.Error("invalid key")
	}
	msg := fmt.Sprintf(cleanTemplate, data.Device, data.Thing, key, value)
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
	deviceSnsCache.Delete(data.Device + key)
	bluedb.DeleteNotice(data.Device, key)
}
