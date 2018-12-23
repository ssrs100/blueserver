package model

import "bluedb"

type ComponentDetail struct {
	ComponentId   string `json:"component_id"`
	ComponentName string `json:"component_name"`
	AdvInterval   int    `json:"adv_interval"`
	TxPower       int    `json:"tx_power"`
	Slot          int    `json:"slot"`
	UpdateStatus  int    `json:"update_status"`
	Data          string `json:"data"`
	UpdateData    string `json:"update_data"`
	NewPassword   string `json:"new_password"`
}

func (b *ComponentDetail) DbObjectTrans(component bluedb.ComponentDetail) ComponentDetail {
	b1 := ComponentDetail{
		ComponentId:   component.Id,
		ComponentName: component.ComponentName,
		AdvInterval:   component.AdvInterval,
		TxPower:       component.TxPower,
		Slot:          component.Slot,
		UpdateStatus:  component.UpdateStatus,
		Data:          component.Data,
		UpdateData:    component.UpdateData,
	}
	return b1
}
