package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const DefaultModalityColor = "#0F766E"

type CatalogStatus string

const (
	CatalogStatusActive   CatalogStatus = "active"
	CatalogStatusArchived CatalogStatus = "archived"
)

type CatalogContext struct {
	Organization Organization
	Unit         Box
}

type Modality struct {
	ID             ID
	OrganizationID ID
	Name           string
	NameKey        string
	Color          string
	Status         CatalogStatus
	ActiveInUnit   bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UnitModality struct {
	OrganizationID ID
	UnitID         ID
	ModalityID     ID
	Active         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Space struct {
	ID              ID
	UnitID          ID
	Name            string
	NameKey         string
	DefaultCapacity *int
	Status          CatalogStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CatalogAuditEvent struct {
	ID             ID
	OrganizationID ID
	UnitID         ID
	ActorUserID    ID
	Action         string
	EntityType     string
	EntityID       ID
	BeforeData     map[string]any
	AfterData      map[string]any
	CreatedAt      time.Time
}

var modalityColorPattern = regexp.MustCompile(`^#[0-9A-F]{6}$`)

func NormalizeCatalogName(value string) (name, key string, valid bool) {
	name = strings.Join(strings.Fields(value), " ")
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return "", "", false
	}
	return name, strings.ToLower(name), true
}

func NormalizeModalityColor(value string) (string, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return DefaultModalityColor, true
	}
	return value, modalityColorPattern.MatchString(value)
}

func ValidCatalogStatus(status CatalogStatus) bool {
	return status == CatalogStatusActive || status == CatalogStatusArchived
}
