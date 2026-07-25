package entities

import (
	"time"

	"github.com/google/uuid"
)

// AdminAuditLog records account changes made through the admin portal.
type AdminAuditLog struct {
	ID           uuid.UUID `json:"id" gorm:"primaryKey;type:uuid"`
	AdminUserID  UserID    `json:"admin_user_id" gorm:"index"`
	AdminEmail   string    `json:"admin_email"`
	Action       string    `json:"action" gorm:"index"`
	TargetUserID UserID    `json:"target_user_id" gorm:"index"`
	Details      string    `json:"details"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}
