package model

import "time"

type Installation struct {
	ID             string    `bson:"_id"             json:"id"`
	InstallationID int64     `bson:"installation_id" json:"installation_id"`
	UserID         string    `bson:"user_id"         json:"user_id"`
	AccountLogin   string    `bson:"account_login"   json:"account_login"`
	AppID          string    `bson:"app_id"          json:"app_id"`
	CreatedAt      time.Time `bson:"created_at"      json:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"      json:"updated_at"`
}
