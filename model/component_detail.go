package model

import (
	"github.com/ssrs100/blueserver/bluedb"
)

const (
	UpdateSuccess = iota
	Updating
	Cancelled
)

type ComponentDetail struct {
	ComponentId  string `json:"component_id"`
	UpdateStatus int    `json:"update_status"`
	Data         string `json:"data"`
	UpdateData   string `json:"update_data"`
	Password     string `json:"passwd"`
}

func (b *ComponentDetail) DbObjectTrans(component bluedb.ComponentDetail) ComponentDetail {
	b1 := ComponentDetail{
		ComponentId:  component.Id,
		UpdateStatus: component.UpdateStatus,
		Data:         component.Data,
		UpdateData:   component.UpdateData,
	}
	return b1
}
