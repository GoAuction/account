package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Profile struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid"`
	UserID        uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	Name          string         `json:"name" gorm:"type:varchar(255);not null"`
	Phone         string         `json:"phone" gorm:"type:varchar(255);default:null"`
	Country       string         `json:"country" gorm:"type:varchar(255);default:null"`
	City          string         `json:"city" gorm:"type:varchar(255);default:null"`
	Description   string         `json:"description" gorm:"type:text;default:null"`
	PhoneVerified bool           `json:"phone_verified" gorm:"type:boolean;not null;default:false"`
	EmailVerified bool           `json:"email_verified" gorm:"type:boolean;not null;default:false"`
	Business      bool           `json:"business" gorm:"type:boolean;not null;default:false"`
	Verified      bool           `json:"verified" gorm:"type:boolean;not null;default:false"`
	Photo         string         `json:"photo" gorm:"type:text;default:null"`
	CreatedAt     time.Time      `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}
