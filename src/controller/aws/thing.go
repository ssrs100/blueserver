package aws

import (
	"bluedb"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/julienschmidt/httprouter"
	"logs"
	"net/http"
	"strconv"
)

var (
	log = logs.GetLogger()

	region = "us-west-1"
)

func ListThings(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	projectId := ps.ByName("projectId")
	u, err := bluedb.QueryUserById(projectId)
	if err != nil {
		log.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("project id not found"))
		return
	}

	if len(u.AccessKey) == 0 || len(u.SecretKey) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("user project id not bind aws"))
		return
	}

	limit := req.URL.Query().Get("limit")
	nextToken := req.URL.Query().Get("nextToken")
	sess := session.Must(session.NewSession())

	// Create a SNS client with additional configuration
	// 方式一: 使用文件证书位置默认为~/.aws/credentials
	//svc := sns.New(sess, aws.NewConfig().WithRegion(region))
	// 方式二: 使用传参方式
	creds := credentials.NewStaticCredentials(
		u.AccessKey,
		u.SecretKey,
		"",
	)
	svc := iot.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})
	awsReq := iot.ListThingsInput{
		NextToken:  nil,
	}
	if len(limit) > 0 {
		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			log.Error("limit is invalid")
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

	rsp, err := svc.ListThings(&awsReq)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		errStr := fmt.Sprintf("aws return err:%s", err.Error())
		_, _ = w.Write([]byte(errStr))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(rsp.String()))
}
