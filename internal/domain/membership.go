package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type MembershipPlanStatus string
type MembershipPlanVersionStatus string
type MembershipBillingCycle string
type MembershipValidityUnit string
type MembershipAccessType string
type MembershipPeriodUnit string
type MembershipDeliveryMode string

const (
	MembershipPlanActive   MembershipPlanStatus = "active"
	MembershipPlanArchived MembershipPlanStatus = "archived"

	MembershipVersionDraft     MembershipPlanVersionStatus = "draft"
	MembershipVersionPublished MembershipPlanVersionStatus = "published"
	MembershipVersionRetired   MembershipPlanVersionStatus = "retired"

	MembershipBillingOneTime    MembershipBillingCycle = "one_time"
	MembershipBillingMonthly    MembershipBillingCycle = "monthly"
	MembershipBillingQuarterly  MembershipBillingCycle = "quarterly"
	MembershipBillingSemiannual MembershipBillingCycle = "semiannual"
	MembershipBillingAnnual     MembershipBillingCycle = "annual"

	MembershipValidityDay   MembershipValidityUnit = "day"
	MembershipValidityWeek  MembershipValidityUnit = "week"
	MembershipValidityMonth MembershipValidityUnit = "month"
	MembershipValidityYear  MembershipValidityUnit = "year"

	MembershipAccessOpen         MembershipAccessType   = "open_access"
	MembershipAccessPerPeriod    MembershipAccessType   = "sessions_per_period"
	MembershipAccessClassPack    MembershipAccessType   = "class_pack"
	MembershipPeriodDay          MembershipPeriodUnit   = "day"
	MembershipPeriodWeek         MembershipPeriodUnit   = "week"
	MembershipPeriodMonth        MembershipPeriodUnit   = "month"
	MembershipDeliveryGroup      MembershipDeliveryMode = "group"
	MembershipDeliveryIndividual MembershipDeliveryMode = "individual"
)

type MembershipPlan struct {
	ID             ID
	OrganizationID ID
	HomeUnitID     ID
	Name           string
	NameKey        string
	Status         MembershipPlanStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Published      *MembershipPlanVersion
	Draft          *MembershipPlanVersion
	Versions       []MembershipPlanVersion
}

type MembershipPlanVersion struct {
	ID             ID
	OrganizationID ID
	PlanID         ID
	VersionNumber  int
	Revision       int
	Status         MembershipPlanVersionStatus
	Name           string
	Description    string
	PriceCents     int
	Currency       string
	BillingCycle   MembershipBillingCycle
	ValidityCount  *int
	ValidityUnit   *MembershipValidityUnit
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Entitlements   []MembershipPlanEntitlement
	Summary        string
	Warnings       []string
}

type MembershipPlanEntitlement struct {
	ID                ID
	OrganizationID    ID
	PlanVersionID     ID
	ModalityID        ID
	ServiceOfferingID ID
	TargetName        string
	TargetType        string
	AccessType        MembershipAccessType
	Quantity          *int
	PeriodUnit        *MembershipPeriodUnit
	DailyLimit        *int
	DeliveryMode      *MembershipDeliveryMode
	BookingRequired   bool
	Position          int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CatalogPlanReference struct {
	ID           ID
	Type         string
	Name         string
	Status       CatalogStatus
	ActiveInUnit bool
}

type MembershipCommandRequest struct {
	ID             ID
	OrganizationID ID
	Command        string
	IdempotencyKey string
	Fingerprint    string
	PlanID         ID
	PlanVersionID  ID
	ResponseData   map[string]any
	CreatedAt      time.Time
}

type MembershipPlanAuditEvent struct {
	ID             ID
	OrganizationID ID
	UnitID         ID
	ActorUserID    ID
	Action         string
	PlanID         ID
	PlanVersionID  ID
	BeforeData     map[string]any
	AfterData      map[string]any
	CreatedAt      time.Time
}

func NormalizeMembershipPlanName(value string) (string, string, bool) {
	name := strings.Join(strings.Fields(value), " ")
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return "", "", false
	}
	return name, strings.ToLower(name), true
}

func NormalizeMembershipDescription(value string) (string, bool) {
	value = strings.TrimSpace(value)
	return value, utf8.RuneCountInString(value) <= 500
}

func ValidMembershipVersion(version MembershipPlanVersion) bool {
	if _, _, ok := NormalizeMembershipPlanName(version.Name); !ok || version.PriceCents <= 0 || version.Currency != "BRL" {
		return false
	}
	if _, ok := NormalizeMembershipDescription(version.Description); !ok {
		return false
	}
	switch version.BillingCycle {
	case MembershipBillingOneTime:
		if version.ValidityCount == nil || version.ValidityUnit == nil || !validValidity(*version.ValidityCount, *version.ValidityUnit) {
			return false
		}
	case MembershipBillingMonthly, MembershipBillingQuarterly, MembershipBillingSemiannual, MembershipBillingAnnual:
		if version.ValidityCount != nil || version.ValidityUnit != nil {
			return false
		}
	default:
		return false
	}
	if len(version.Entitlements) == 0 {
		return false
	}
	seen := map[string]struct{}{}
	for _, entitlement := range version.Entitlements {
		if !ValidMembershipEntitlement(entitlement) {
			return false
		}
		key := string(entitlement.ModalityID) + ":" + string(entitlement.ServiceOfferingID)
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func ValidMembershipEntitlement(value MembershipPlanEntitlement) bool {
	if (value.ModalityID == "") == (value.ServiceOfferingID == "") || value.Position < 0 {
		return false
	}
	if value.DailyLimit != nil && (*value.DailyLimit < 1 || *value.DailyLimit > 10000) {
		return false
	}
	if value.DeliveryMode != nil && *value.DeliveryMode != MembershipDeliveryGroup && *value.DeliveryMode != MembershipDeliveryIndividual {
		return false
	}
	switch value.AccessType {
	case MembershipAccessOpen:
		return value.Quantity == nil && value.PeriodUnit == nil
	case MembershipAccessPerPeriod:
		return validQuantity(value.Quantity) && value.PeriodUnit != nil && (*value.PeriodUnit == MembershipPeriodDay || *value.PeriodUnit == MembershipPeriodWeek || *value.PeriodUnit == MembershipPeriodMonth)
	case MembershipAccessClassPack:
		return validQuantity(value.Quantity) && value.PeriodUnit == nil
	default:
		return false
	}
}

func MembershipPlanSummary(version MembershipPlanVersion) string {
	targets := make([]string, 0, len(version.Entitlements))
	rules := make([]string, 0, len(version.Entitlements))
	for _, entitlement := range version.Entitlements {
		targets = append(targets, entitlement.TargetName)
		rules = append(rules, entitlementSummary(entitlement))
	}
	targetText := strings.Join(targets, ", ")
	ruleText := strings.Join(uniqueStrings(rules), ", ")
	if version.BillingCycle == MembershipBillingOneTime && version.ValidityCount != nil && version.ValidityUnit != nil {
		ruleText += fmt.Sprintf(" por %d %s", *version.ValidityCount, validityLabel(*version.ValidityCount, *version.ValidityUnit))
	}
	return strings.Join([]string{targetText, ruleText, membershipPriceSummary(version)}, " · ")
}

func membershipPriceSummary(version MembershipPlanVersion) string {
	reais := fmt.Sprintf("R$ %d", version.PriceCents/100)
	if version.PriceCents%100 != 0 {
		reais = fmt.Sprintf("R$ %d,%02d", version.PriceCents/100, version.PriceCents%100)
	}
	switch version.BillingCycle {
	case MembershipBillingMonthly:
		return reais + "/mês"
	case MembershipBillingQuarterly:
		return reais + "/trimestre"
	case MembershipBillingSemiannual:
		return reais + "/semestre"
	case MembershipBillingAnnual:
		return reais + "/ano"
	default:
		return reais + " pagamento único"
	}
}

func entitlementSummary(value MembershipPlanEntitlement) string {
	switch value.AccessType {
	case MembershipAccessPerPeriod:
		return fmt.Sprintf("%d sessões por %s", pointerInt(value.Quantity), periodLabel(value.PeriodUnit))
	case MembershipAccessClassPack:
		return fmt.Sprintf("pacote de %d aulas", pointerInt(value.Quantity))
	default:
		return "acesso livre"
	}
}

func validValidity(count int, unit MembershipValidityUnit) bool {
	switch unit {
	case MembershipValidityDay:
		return count >= 1 && count <= 3650
	case MembershipValidityWeek:
		return count >= 1 && count <= 520
	case MembershipValidityMonth:
		return count >= 1 && count <= 120
	case MembershipValidityYear:
		return count >= 1 && count <= 10
	default:
		return false
	}
}

func validQuantity(value *int) bool { return value != nil && *value >= 1 && *value <= 10000 }
func pointerInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
func periodLabel(value *MembershipPeriodUnit) string {
	if value == nil {
		return "período"
	}
	switch *value {
	case MembershipPeriodDay:
		return "dia"
	case MembershipPeriodWeek:
		return "semana"
	default:
		return "mês"
	}
}
func validityLabel(count int, unit MembershipValidityUnit) string {
	if count == 1 {
		switch unit {
		case MembershipValidityDay:
			return "dia"
		case MembershipValidityWeek:
			return "semana"
		case MembershipValidityMonth:
			return "mês"
		default:
			return "ano"
		}
	}
	switch unit {
	case MembershipValidityDay:
		return "dias"
	case MembershipValidityWeek:
		return "semanas"
	case MembershipValidityMonth:
		return "meses"
	default:
		return "anos"
	}
}
func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}
