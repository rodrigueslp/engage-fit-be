package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

const DefaultUnitTimezone = "America/Sao_Paulo"

type UnitAccessContext struct {
	Organization Organization
	Unit         Box
	Role         UserRole
}

type OrganizationAuditEvent struct {
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

func NormalizeUnitName(value string) (string, string, bool) {
	visible := strings.Join(strings.Fields(value), " ")
	if visible == "" || utf8.RuneCountInString(visible) > 120 {
		return "", "", false
	}
	return visible, strings.ToLower(visible), true
}
