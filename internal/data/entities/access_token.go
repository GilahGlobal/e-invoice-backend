package entities

import "time"

type AccessToken struct {
	ID                        string    `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	OwnerID                   string    `gorm:"column:owner_id; type:uuid; not null" json:"owner_id"`
	IsLive                    bool      `gorm:"column:is_live; type:bool; default:false; not null" json:"is_live"`
	LoginAccessToken          string    `gorm:"column:login_access_token; type:text" json:"-"`
	LoginAccessTokenExpiresIn string    `gorm:"column:login_access_token_expires_in; type:varchar(250)" json:"-"`
	CreatedAt                 time.Time `gorm:"column:created_at; autoCreateTime" json:"created_at"`
	UpdatedAt                 time.Time `gorm:"column:updated_at; autoUpdateTime" json:"updated_at"`
}
