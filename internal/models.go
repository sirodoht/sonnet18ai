package internal

import (
	"time"
)

type (
	Document struct {
		DocumentID string `gorm:"type:varchar(255);primary_key"`
		Body       string `gorm:"type:text"`
		CreatedAt  time.Time
		UpdatedAt  time.Time
	}
)
