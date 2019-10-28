package aws

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/bluedb"
	"io/ioutil"
	"net/http"
	"strings"
)

type UpdateCertReq struct {
	ThingName []string `json:"thing_names"`
	Cert      string   `json:"cert"`
}

func UpdateThingCert(w http.ResponseWriter, req *http.Request, ps map[string]string) {
	projectId := ps["projectId"]
	u, err := bluedb.QueryUserById(projectId)
	if err != nil {
		logs.Error("Invalid projectId. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("project id not found"))
		return
	}
	if len(u.AccessKey) == 0 || len(u.SecretKey) == 0 {
		logs.Info("%s ak/sk is empty, ready to create", u.Name)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logs.Error("Receive body failed: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer req.Body.Close()

	var certReq = UpdateCertReq{}
	if err = json.Unmarshal(body, &certReq); err != nil {
		logs.Error("Invalid body. err:%s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	//// check thing name
	//if len(certReq.ThingName) == 0 {
	//	errStr := fmt.Sprintf("thing name is empty.")
	//	w.WriteHeader(http.StatusBadRequest)
	//	_, _ = w.Write([]byte(errStr))
	//	return
	//}

	// create thing
	logs.Info("update thing cert..")
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(
		u.AccessKey,
		u.SecretKey,
		"",
	)

	svc := iot.New(sess, &aws.Config{Credentials: creds, Region: aws.String(region)})
	certUrn := bluedb.GetSys("certUrn")
	certId := strings.Split(certUrn, "/")[1]
	descReq := iot.DescribeCertificateInput{
		CertificateId: &certId,
	}
	out, err := svc.DescribeCertificate(&descReq)
	if err != nil {
		logs.Error("describe principal err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	logs.Info("start cert====>")
	logs.Info(*out.CertificateDescription.CertificatePem)
	logs.Info("end cert====>")

	isActive := true
	createCertReq := iot.CreateKeysAndCertificateInput{
		SetAsActive: &isActive,
	}
	outC, err := svc.CreateKeysAndCertificate(&createCertReq)
	if err != nil {
		logs.Error("create cert err:%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	logs.Info("create cert pem====>")
	logs.Info(*outC.CertificatePem)
	logs.Info("create cert private====>")
	logs.Info(*outC.KeyPair.PrivateKey)
	logs.Info("create cert public====>")
	logs.Info(*outC.KeyPair.PublicKey)
	w.WriteHeader(http.StatusOK)
}
