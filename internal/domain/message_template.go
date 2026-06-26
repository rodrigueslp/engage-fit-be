package domain

import "time"

type MessageTemplate struct {
	ID         ID
	BoxID      ID
	Name       string
	Content    string
	ContentSID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
