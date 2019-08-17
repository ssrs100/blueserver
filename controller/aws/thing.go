package aws

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"github.com/ssrs100/blueserver/influxdb"
	"net/http"
	"strconv"
)

var (
	region = "us-west-2"
)

func GetThingLatestData(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	thing := ps["thingName"]
	data, err := influxdb.GetLatest("temperature", thing)
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
		NextToken: nil,
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
		awsReq = iot.ListThingsInput{
			MaxResults: &limit64,
		}
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
	//_, _ = w.Write([]byte(rsp.String()))
	_, _ = w.Write(outBytes)
}

func listThings(svc *iot.IoT, input *iot.ListThingsInput) (*iot.ListThingsOutput, error) {
	req, out := svc.ListThingsRequest(input)
	req.HTTPRequest.Header.Add("Accept", "application/json")
	return out, req.Send()
}
