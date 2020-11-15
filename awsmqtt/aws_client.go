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
	"github.com/ssrs100/blueserver/sesscache"
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

var _ aws.Config
var _ awserr.Error
var _ request.Request

type snsSend struct {
	key        string
	data       *influxdb.ReportData
	upperLimit bool
	isClean    bool
}

type AwsIotClient struct {
	reportChan chan *Shadow
	awsClient  *Client
	snsClient  *sns.SNS
	user       *bluedb.User

	snsChan chan *snsSend
}

type thresh struct {
	minTemp float32
	maxTemp float32
	minHum  float32
	maxHum  float32
}

const (
	tempKey     = "temperature"
	humidityKey = "humidity"
)

var msgTemplate = "[notice]device(%s) thing(%s) %s is %v, it's out of the range of device settings, please pay attention to it."

var cleanTemplate = "[clean]device(%s) thing(%s) %s is %v, it restores back to the range of device settings."

var (
	cleanCache     *cache.Cache
	useClientCache map[string]*AwsIotClient
)

var stopChan chan interface{}

var defaultThresh thresh

func init() {
	cleanCache = cache.New(time.Minute, 2*time.Minute)
	useClientCache = make(map[string]*AwsIotClient)

	tempMinThresh := conf.GetFloatWithDefault("temperature_min_thresh", common.MinTemp)
	tempMaxThresh := conf.GetFloatWithDefault("temperature_max_thresh", common.MaxTemp)
	humiMinThresh := conf.GetFloatWithDefault("humi_min_thresh", common.MinHumi)
	humiMaxThresh := conf.GetFloatWithDefault("humi_max_thresh", common.MaxHumi)
	defaultThresh = thresh{
		minTemp: float32(tempMinThresh),
		maxTemp: float32(tempMaxThresh),
		minHum:  float32(humiMinThresh),
		maxHum:  float32(humiMaxThresh),
	}
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

	ts := thingStatus{}
	ts.Init(stopChan)

	for _, u := range users {
		user, err := bluedb.QueryUserByName(u)
		if err != nil {
			logs.Error("get user(%s) fail, err:%s", err.Error())
			continue
		}
		if user == nil || len(user.AccessKey) == 0 || len(user.SecretKey) == 0 {
			logs.Error("get user(%s) fail, or no ak sk set", u)
			continue
		}
		c := conf.LoadFile(filepath.Join(certDir, u, "conf.json"))
		if c == nil {
			logs.Error("load user(%s) conf.json fail", u)
			continue
		}
		iotEndpoint := c.GetString("iot_endpoint")
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
		awsIC.user = user
		awsIC.reportChan, err = awsIC.awsClient.SubscribeForThingReport()
		if err != nil {
			logs.Error("subscribe user(%s) thing report fail", u)
			continue
		}
		awsIC.snsChan = make(chan *snsSend, 200)
		go awsIC.startAwsClient(user.Id, stopChan)
		awsIC.initSns()
		go awsIC.sendSns()
		useClientCache[u] = &awsIC
	}
	logs.Info("start aws client success")
	<-stopChan
}

func (ac *AwsIotClient) publishEcho() {
	topic := "$aws/things/" + common.TestThing + "/echo"
	res := ac.awsClient.client.Publish(topic, 0, false, "")
	if res.WaitTimeout(time.Second*5) && res.Error() != nil {
		log.Fatal("no report.json found", res.Error())
	}
}

func (ac *AwsIotClient) startAwsClient(projectId string, stop chan interface{}) {
	for {
		select {
		case s, ok := <-ac.reportChan:
			if !ok {
				logs.Debug("failed to read from shadow channel")
			} else {
				var rds []*influxdb.ReportData
				logs.Info("rcv thing:%s", s.Thing)
				if s.Thing == common.TestThing {
					// for report check
					go ac.publishEcho()
					continue
				}
				logs.Debug("%s", string(s.Msg))

				// set thing status
				var dbThing *bluedb.Thing
				thing := s.Thing
				if dbThing = bluedb.GetThingByName(thing); dbThing == nil {
					logs.Info("thing(%s) not register, ignore", thing)
					if _, ok := cleanCache.Get(thing); !ok {
						go ac.stopThing(thing)
					} else {
						logs.Info("already send stop, wait cache timeout")
					}
					continue
				}
				sesscache.SetWithExpired(common.StatusKey(thing), OnLine, 5*time.Minute)
				if strconv.Itoa(dbThing.Status) != OnLine {
					dbThing.Status = 1
					bluedb.UpdateThingStatus(*dbThing)
				}

				// save data
				rdList := influxdb.ReportDataList{}
				if err := json.Unmarshal(s.Msg, &rdList); err != nil {
					logs.Error("err:%s, msg:%s", err.Error(), string(s.Msg))
					continue
				}
				if len(rdList.Objects) == 0 {
					rd := influxdb.ReportData{}
					if err := json.Unmarshal(s.Msg, &rd); err != nil {
						logs.Error("err:%s, msg:%s", err.Error(), string(s.Msg))
						continue
					}
					rd.Thing = dbThing.Name
					rd.ProjectId = dbThing.ProjectId
					rds = append(rds, &rd)
				} else {
					for _, r := range rdList.Objects {
						r.Thing = dbThing.Name
						r.ProjectId = dbThing.ProjectId
						rds = append(rds, r)
					}
				}

				var sensorList, beaconList []*influxdb.ReportData
				for _, r := range rds {
					if r.DataType == "broadcast" {
						beaconList = append(beaconList, r)
					} else {
						sensorList = append(sensorList, r)
					}
				}
				if err := influxdb.InsertSensorData(influxdb.TableTemperature, sensorList); err != nil {
					logs.Error("%s", err.Error())
				}
				if err := influxdb.InsertBeaconData(influxdb.TableBroadcast, beaconList); err != nil {
					logs.Error("%s", err.Error())
				}
				for _, r := range rds {
					if r.DataType != "broadcast" {
						ac.processOneRdMessage(r)
					}
				}
				//logs.Debug("insert %v", rd)
				//logs.Debug("insert influxdb success")
			}
		case <-stop:
			close(ac.snsChan)
			logs.Info("stopped")
		}
	}
}

func (ac *AwsIotClient) processOneRdMessage(rd *influxdb.ReportData) {
	var tmp float32
	tmpFloat, err := strconv.ParseFloat(string(rd.Temperature), 64)
	if err != nil {
		tmpInt, err := strconv.Atoi(string(rd.Temperature))
		if err != nil {
			logs.Error("temper err:%v", rd.Temperature)
			return
		}
		tmp = float32(tmpInt)
	} else {
		tmp = float32(tmpFloat)
	}

	// humidity
	var hum float32
	humFloat, err := strconv.ParseFloat(string(rd.Humidity), 64)
	if err != nil {
		humInt, err := strconv.Atoi(string(rd.Humidity))
		if err != nil {
			logs.Error("humidity err:%v", rd.Humidity)
			return
		}
		hum = float32(humInt)
	} else {
		hum = float32(humFloat)
	}

	threshDevice := getThresh(rd, &defaultThresh)
	if tmp >= threshDevice.maxTemp {
		ac.snsChan <- &snsSend{key: tempKey, data: rd, upperLimit: true, isClean: false}
		//go ac.sendSns(tempKey, rd, true, false)
	} else if tmp < threshDevice.minTemp {
		ac.snsChan <- &snsSend{key: tempKey, data: rd, upperLimit: false, isClean: false}
		//go ac.sendSns(tempKey, rd, false, false)
	} else {
		ac.snsChan <- &snsSend{key: tempKey, data: rd, upperLimit: false, isClean: true}
		//go ac.sendSns(tempKey, rd, false, true)
	}

	// humidity
	if hum >= threshDevice.maxHum {
		ac.snsChan <- &snsSend{key: humidityKey, data: rd, upperLimit: true, isClean: false}
		//go ac.sendSns(humidityKey, rd, true, false)
	} else if hum < threshDevice.minHum {
		ac.snsChan <- &snsSend{key: humidityKey, data: rd, upperLimit: false, isClean: false}
		//go ac.sendSns(humidityKey, rd, false, false)
	} else {
		ac.snsChan <- &snsSend{key: humidityKey, data: rd, upperLimit: false, isClean: true}
		//go ac.sendSns(humidityKey, rd, false, true)
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
	if ac.snsClient == nil {
		logs.Error("init sns user err:%s", ac.user.Name)
	}
}

func (ac *AwsIotClient) sendSns() {
	for {
		send, opened := <-ac.snsChan
		if !opened {
			logs.Info("sns chan closed")
			return
		}
		cause := "upper"
		if !send.upperLimit {
			cause = "lower"
		}
		if send.isClean {
			ac.sendCleanMsg(send.key, send.data)
		} else {
			ac.sendNotifyMsg(cause, send.key, send.data)
		}
	}

}

func (ac *AwsIotClient) sendNotifyMsg(cause, key string, data *influxdb.ReportData) {
	defer func() {
		if p := recover(); p != nil {
			logs.Error("panic err:%v", p)
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			logs.Error("==> %s\n", string(buf[:n]))
		}
	}()
	noticeKey := common.NoticeKey(data.ProjectId, data.Device+key+cause)
	noticeVal := sesscache.Get(noticeKey)
	if len(noticeVal) > 0 {
		return
	}
	logs.Debug("send %s %s start", key, cause)
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
		return
	}
	msg := fmt.Sprintf(msgTemplate, data.Device, data.Thing, key, value)

	// send to app
	devs := bluedb.QueryDevToken(data.ProjectId)
	devTokens := make([]string, 0)
	for _, dts := range devs {
		devTokens = append(devTokens, dts.DeviceToken)
	}
	NotifyApp(devTokens, msg)

	// send to sns
	params := &sns.PublishInput{
		Message:  aws.String(msg),
		TopicArn: aws.String(fmt.Sprintf("arn:aws:sns:us-west-2:415890359503:%s", data.ProjectId)),
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
		ProjectId: data.ProjectId,
		Device:    data.Device,
		Noticed:   "1",
		Key:       key,
		Cause:     cause,
	}
	sesscache.SetWithExpired(noticeKey, "1", 24*time.Hour)
	d, _ := bluedb.QueryNoticeByDeviceWithCause(data.ProjectId, data.Device, key, cause)
	if d == nil {
		bluedb.SaveNotice(n)
	}
}

func (ac *AwsIotClient) sendCleanMsg(key string, data *influxdb.ReportData) {
	defer func() {
		if p := recover(); p != nil {
			logs.Error("panic err:%v", p)
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			logs.Error("==> %s\n", string(buf[:n]))
		}
	}()
	upCause := "upper"
	upKey := common.NoticeKey(data.ProjectId, data.Device+key+upCause)
	upVal := sesscache.Get(upKey)

	lwCause := "lower"
	lwKey := common.NoticeKey(data.ProjectId, data.Device+key+lwCause)
	lwVal := sesscache.Get(lwKey)
	if len(upVal) == 0 && len(lwVal) == 0 {
		// not send
		d, _ := bluedb.QueryNoticeByDevice(data.ProjectId, data.Device, key)
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
		return
	}
	msg := fmt.Sprintf(cleanTemplate, data.Device, data.Thing, key, value)

	// send to app
	devs := bluedb.QueryDevToken(data.ProjectId)
	devTokens := make([]string, 0)
	for _, dts := range devs {
		devTokens = append(devTokens, dts.DeviceToken)
	}
	NotifyApp(devTokens, msg)

	// send to sns
	params := &sns.PublishInput{
		Message:  aws.String(msg),
		TopicArn: aws.String(fmt.Sprintf("arn:aws:sns:us-west-2:415890359503:%s", data.ProjectId)),
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
	sesscache.Del(upKey)
	sesscache.Del(lwKey)
	if err := bluedb.DeleteNotice(data.ProjectId, data.Device, key); err != nil {
		logs.Error("delete notice err:%s", err.Error())
	}
}

func (ac *AwsIotClient) stopThing(thing string) {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("panic err:%v", r)
			var buf [4096]byte
			n := runtime.Stack(buf[:], true)
			logs.Error("==> %s\n", string(buf[:n]))
		}
	}()
	logs.Info("stop thing:%s", thing)
	stopTopic := fmt.Sprintf("$aws/things/%s/reports/stop", thing)
	res := ac.awsClient.client.Publish(stopTopic, 0, false, []byte("stop report"))
	if res.WaitTimeout(time.Second*2) && res.Error() != nil {
		logs.Error("stop fail, err:%s", res.Error().Error())
	}
	cleanCache.Set(thing, "", cache.DefaultExpiration)
}

func (ac *AwsIotClient) startThing(thing string) error {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("panic err:%v", r)
			var buf [4096]byte
			n := runtime.Stack(buf[:], true)
			logs.Error("==> %s\n", string(buf[:n]))
		}
	}()
	logs.Info("start thing:%s", thing)
	stopTopic := fmt.Sprintf("$aws/things/%s/reports/start", thing)
	res := ac.awsClient.client.Publish(stopTopic, 0, false, []byte("start report"))
	if res.WaitTimeout(time.Second*2) && res.Error() != nil {
		logs.Error("stop fail, err:%s", res.Error().Error())
		return res.Error()
	}
	return nil
}
