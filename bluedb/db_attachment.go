package bluedb

import (
	"errors"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // import your used driver
	"github.com/jack0liu/logs"
	"github.com/satori/go.uuid"
	"time"
)

type Attachment struct {
	Id             string    `orm:"size(64);pk"`
	BeaconId       string    `orm:"size(128)"`
	AttachmentName string    `orm:"size(64)"`
	AttachmentType string    `orm:"size(64)";null`
	Data           string    `orm:"type(text);null"`
	Create_at      time.Time `orm:"auto_now_add;type(datetime)"`
}

func init() {
	orm.RegisterModel(new(Attachment))
}

func CreateAttachment(attachment Attachment) string {
	o := orm.NewOrm()
	u2 := uuid.NewV4()
	attachment.Id = u2.String()
	// insert
	_, err := o.Insert(&attachment)
	if err != nil {
		logs.Error("create attachment fail.attachment: %v", attachment)
	}
	logs.Info("create beacon id: %v", attachment.Id)
	return attachment.Id
}

func DeleteAttachment(id string) error {
	o := orm.NewOrm()
	b := Attachment{Id: id}
	if _, err := o.Delete(&b); err != nil {
		return err
	}
	logs.Info("delete attachment: %v", id)
	return nil
}

func DeleteAttachmentByBeacon(beaconId string) error {
	o := orm.NewOrm()
	r := o.Raw("DELETE from attachment WHERE beacon_id = ?", beaconId)
	if _, err := r.Exec(); err != nil {
		logs.Error("delete beaconId(%s) attachment failed, cause:%s", beaconId, err.Error())
		return errors.New("Deleting attachment by beacon id failed.")
	}
	logs.Info("delete beaconId(%s) attachment success", beaconId)
	return nil
}

func QueryAttachments(params map[string]interface{}) []Attachment {
	var attachments []Attachment
	o := orm.NewOrm()
	qs := o.QueryTable("attachment")

	if beaconId, ok := params["beacon_id"]; ok {
		qs = qs.Filter("beacon_id", beaconId)
	}

	if attachTypes, ok := params["attachment_types"]; ok {
		if len(attachTypes.([]string)) > 0 {
			qs = qs.Filter("attachment_type__in", attachTypes)
		}
	}

	if offset, ok := params["offset"]; ok {
		qs = qs.Limit(offset.(int))
	}

	if limit, ok := params["limit"]; ok {
		qs = qs.Limit(limit.(int))
	}

	qs = qs.OrderBy("create_at")
	_, err := qs.All(&attachments)
	if err != nil {
		logs.Error("query attachment fail, err:%s", err.Error())
	}
	return attachments
}

func QueryAttachmentById(id string) (u *Attachment, err error) {
	o := orm.NewOrm()
	attachment := Attachment{Id: id}
	if err := o.Read(&attachment); err != nil {
		logs.Error("query attachment fail: %v", id)
		return nil, err
	}
	return &attachment, nil
}
