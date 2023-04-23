package internal

import (
	"time"
)

type (
	Document struct {
		DocumentID uint   `gorm:"primary_key;AUTO_INCREMENT"`
		Title      string `gorm:"type:varchar(255)"`
		Body       string `gorm:"type:text"`
		CreatedAt  time.Time
		UpdatedAt  time.Time
	}
)
