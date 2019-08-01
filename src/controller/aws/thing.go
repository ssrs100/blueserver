package aws

import (
	"bluedb"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
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
