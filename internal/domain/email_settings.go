package domain

import "time"

type EmailSettings struct {
	ID                ID
	BoxID             ID
	Provider          string
	SMTPHost          string
	SMTPPort          int
	Username          string
	PasswordEncrypted string
	FromEmail         string
	FromName          string
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
