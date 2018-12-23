package controller

import (
	"bluedb"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"mqttclient"
	"net/http"
	"strconv"
	"time"
	"utils"
)

type Component struct {
	Id                string `json:"id"`
	MacAddr           string `json:"mac_addr"`
	Type              string `json:"type"`
	ProjectId         string `json:"project_id"`
	Name              string `json:"name"`
	ComponentPassword string `json:"component_password"`
}

func (b *Component) dbObjectTrans(component bluedb.Component) Component {
	b1 := Component{
		Id:                component.Id,
		MacAddr:           component.MacAddr,
		Type:              component.Type,
		ProjectId:         component.ProjectId,
		Name:              component.Name,
		ComponentPassword: component.ComponentPassword,
	}
	return b1
}

func (b *Component) dbListObjectTrans(components []bluedb.Component) []Component {
	ret := make([]Component, 0)
	for _, v := range components {
		ret = append(ret, b.dbObjectTrans(v))
	}
	return ret
}

type CreateComponentResponse struct {
	ComponentId string `json:"id"`
}

func RegisterComponent(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var componentReq = &Component{}
	err = json.Unmarshal(body, componentReq)
	if err != nil {
		log.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	// check component info
	if componentReq.Type != utils.ComponentBeaconType && componentReq.Type != utils.ComponentGatewayType {
		strErr := fmt.Sprintf("Invalid type(%s).", componentReq.Type)
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}
	projectId := ps.ByName("projectId")
	if len(projectId) == 0 {
		strErr := fmt.Sprintf("Invalid project_id(%s).", projectId)
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	if len(componentReq.Name) <= 0 || len(componentReq.MacAddr) < 10 || len(componentReq.ComponentPassword) <= 0 {
		strErr := fmt.Sprintf("Invalid name(%s) or mac(%s) or password(%s).",
			componentReq.Name, componentReq.MacAddr, componentReq.ComponentPassword)
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	params := make(map[string]interface{})
	params["type"] = componentReq.Type
	params["project_id"] = projectId
	params["mac_addr"] = componentReq.MacAddr
	components := bluedb.QueryComponents(params)
	if len(components) > 0 {
		strErr := fmt.Sprintf("Component(%s) has been registed.", componentReq.MacAddr)
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	// subscribe if type is gateway
	if componentReq.Type == utils.ComponentGatewayType && mqttclient.Client != nil {
		mqttclient.Client.Subscribe(componentReq.MacAddr)
	}

	log.Info("register component:%v", componentReq)
	componentDb := bluedb.Component{
		Id:                "",
		MacAddr:           componentReq.MacAddr,
		Type:              componentReq.Type,
		ProjectId:         projectId,
		Name:              componentReq.Name,
		ComponentPassword: componentReq.ComponentPassword,
		CreateAt:          time.Now(),
	}
	componentId := bluedb.CreateComponent(componentDb)
	if len(componentId) <= 0 {
		strErr := fmt.Sprintf("component(%s) regiter fail.", componentReq.MacAddr)
		log.Error(strErr)
		DefaultHandler.ServeHTTP(w, req, errors.New(strErr), http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateComponentResponse{
		ComponentId: componentId,
	})
	w.WriteHeader(http.StatusOK)
}

func DeleteComponent(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("componentId")

	com, _ := bluedb.QueryComponentById(id)
	// unsubscribe if type is gateway
	if com != nil && com.Type == utils.ComponentGatewayType && mqttclient.Client != nil {
		mqttclient.Client.UnSubscribe(com.MacAddr)
	}

	err := bluedb.DeleteComponent(id)
	if err != nil {
		log.Error("Delete component failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

}

func ListComponents(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	projectId := ps.ByName("projectId")
	params := make(map[string]interface{})
	params["project_id"] = projectId

	if limit, err := strconv.Atoi(req.Form.Get("limit")); err == nil {
		params["limit"] = limit
	}

	if offset, err := strconv.Atoi(req.Form.Get("offset")); err == nil {
		params["offset"] = offset
	}

	if name := req.Form.Get("name"); len(name) > 0 {
		params["name"] = name
	}

	components := bluedb.QueryComponents(params)

	log.Debug("list components:%v", components)
	var b = Component{}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b.dbListObjectTrans(components))
}

func UpdateComponent(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("Receive body failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var comReq = &Component{}
	err = json.Unmarshal(body, comReq)
	if err != nil {
		log.Error("Invalid body. err:%s", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	// check
	id := ps.ByName("componentId")
	if len(id) == 0 {
		log.Error("update component fail, id should be set.")
		DefaultHandler.ServeHTTP(w, req, errors.New("Component id should be set."), http.StatusBadRequest)
		return
	}

	//TODO: check device
	comdb, err := bluedb.QueryComponentById(id)
	if err != nil {
		log.Error("update component fail, id not found.")
		DefaultHandler.ServeHTTP(w, req, errors.New("Component id not found."), http.StatusBadRequest)
		return
	}

	var compass, name string
	if len(comReq.ComponentPassword) == 0 {
		compass = comdb.ComponentPassword
	}

	if len(comReq.Name) == 0 {
		name = comdb.Name
	}

	component := bluedb.Component{
		Id:                id,
		ComponentPassword: compass,
		Name:              name,
	}
	err = bluedb.UpdateComponent(component)
	if err != nil {
		log.Error("Update component failed: %v", err.Error())
		DefaultHandler.ServeHTTP(w, req, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

}
