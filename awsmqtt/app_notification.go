package awsmqtt

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var client = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, netw, addr string) (net.Conn, error) {
			deadline := time.Now().Add(25 * time.Second)
			c, err := net.DialTimeout(netw, addr, time.Second*20)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		},
	},
}

type Aps struct {
	Alert string `json:"alert"`
	Badge int    `json:"badge"`
	Sound string `json:"sound"`
}

type IosPayLoad struct {
	Aps Aps `json:"aps"`
}
type Params struct {
	AppKey       string     `json:"appkey"`
	Timestamp    string     `json:"timestamp"`
	DeviceTokens string     `json:"device_tokens"`
	Type         string     `json:"type"`
	ProductMode  string     `json:"production_mode"`
	Payload      IosPayLoad `json:"payload"`
}

func NotifyApp(deviceToken []string, title string) {
	if len(deviceToken) == 0 {
		return
	}
	method := "POST"
	url := conf.GetString("app_url")
	iosKey := conf.GetString("app_ios_key")
	iosMasterKey := conf.GetString("app_ios_master_key")

	devs := strings.Join(deviceToken, ",")

	ipl := IosPayLoad{
		Aps: Aps{
			Alert: title,
			Sound: "default",
			Badge: 1,
		},
	}
	now := time.Now().Unix()
	params := Params{
		AppKey:       iosKey,
		Timestamp:    strconv.Itoa(int(now)),
		DeviceTokens: devs,
		Type:         "listcast",
		ProductMode:  "false",
		Payload:      ipl,
	}

	reqBody, err := json.Marshal(&params)
	if err != nil {
		logs.Error("marshal fail, err:%s", err.Error())
		return
	}
	logs.Debug("reqBody:%s", string(reqBody))
	urlSend := fmt.Sprintf("%s/api/send", url)
	signRaw := fmt.Sprintf("%s%s%s%s", method, urlSend, string(reqBody), iosMasterKey)

	// sign request
	m := md5.New()
	_, _ = m.Write([]byte(signRaw))
	out := m.Sum(nil)
	urlSend = urlSend + "?sign=" + hex.EncodeToString(out)

	req, err := http.NewRequest(method, urlSend, bytes.NewReader(reqBody))
	if err != nil {
		logs.Error("new request fail, err:%s", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		logs.Error("client do fail, err:%s", err.Error())
		return
	}
	defer res.Body.Close()
	respBody, err := ioutil.ReadAll(res.Body)
	logs.Debug("respCode:%d, respBody:%s", res.StatusCode, string(respBody))
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		logs.Error("respCode:%d, respBody:%s", res.StatusCode, string(respBody))
	}
}
