package domain

import "time"

type EmailTemplate struct {
	ID        ID
	BoxID     ID
	Name      string
	Subject   string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
