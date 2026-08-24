package domain

import "testing"

func TestNormalizeMembershipPlanName(t *testing.T) {
	name, key, ok := NormalizeMembershipPlanName("  Pilates   Duas Vezes ")
	if !ok || name != "Pilates Duas Vezes" || key != "pilates duas vezes" {
		t.Fatalf("unexpected normalization: %q %q %v", name, key, ok)
	}
}

func TestMembershipEntitlementValidationMatrix(t *testing.T) {
	quantity, daily, zero, tooMany := 2, 1, 0, 10001
	day, week, month := MembershipPeriodDay, MembershipPeriodWeek, MembershipPeriodMonth
	group := MembershipDeliveryGroup
	invalidDelivery := MembershipDeliveryMode("remote")
	tests := []struct {
		name  string
		value MembershipPlanEntitlement
		valid bool
	}{
		{"open modality", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen}, true},
		{"open with daily limit", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen, DailyLimit: &daily}, true},
		{"open with quantity", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen, Quantity: &quantity}, false},
		{"open with period", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen, PeriodUnit: &week}, false},
		{"daily sessions", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessPerPeriod, Quantity: &quantity, PeriodUnit: &day}, true},
		{"weekly sessions", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessPerPeriod, Quantity: &quantity, PeriodUnit: &week, DailyLimit: &daily, DeliveryMode: &group}, true},
		{"monthly sessions", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessPerPeriod, Quantity: &quantity, PeriodUnit: &month}, true},
		{"sessions missing period", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessPerPeriod, Quantity: &quantity}, false},
		{"sessions zero quantity", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessPerPeriod, Quantity: &zero, PeriodUnit: &week}, false},
		{"sessions excessive quantity", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessPerPeriod, Quantity: &tooMany, PeriodUnit: &week}, false},
		{"class pack service", MembershipPlanEntitlement{ServiceOfferingID: "s1", AccessType: MembershipAccessClassPack, Quantity: &quantity}, true},
		{"class pack with period", MembershipPlanEntitlement{ServiceOfferingID: "s1", AccessType: MembershipAccessClassPack, Quantity: &quantity, PeriodUnit: &month}, false},
		{"zero daily limit", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen, DailyLimit: &zero}, false},
		{"excessive daily limit", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen, DailyLimit: &tooMany}, false},
		{"invalid delivery", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen, DeliveryMode: &invalidDelivery}, false},
		{"negative position", MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen, Position: -1}, false},
		{"two targets", MembershipPlanEntitlement{ModalityID: "m1", ServiceOfferingID: "s1", AccessType: MembershipAccessOpen}, false},
		{"no target", MembershipPlanEntitlement{AccessType: MembershipAccessOpen}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := ValidMembershipEntitlement(test.value); actual != test.valid {
				t.Fatalf("expected %v, got %v", test.valid, actual)
			}
		})
	}
}

func TestMembershipVersionBillingAndValidityMatrix(t *testing.T) {
	base := MembershipPlanVersion{Name: "Pilates", PriceCents: 26000, Currency: "BRL", BillingCycle: MembershipBillingMonthly, Entitlements: []MembershipPlanEntitlement{{ModalityID: "m1", AccessType: MembershipAccessOpen}}}
	if !ValidMembershipVersion(base) {
		t.Fatal("expected monthly version to be valid")
	}
	count, unit := 90, MembershipValidityDay
	oneTime := base
	oneTime.BillingCycle, oneTime.ValidityCount, oneTime.ValidityUnit = MembershipBillingOneTime, &count, &unit
	if !ValidMembershipVersion(oneTime) {
		t.Fatal("expected one-time validity to be valid")
	}
	invalidRecurring := base
	invalidRecurring.ValidityCount, invalidRecurring.ValidityUnit = &count, &unit
	if ValidMembershipVersion(invalidRecurring) {
		t.Fatal("recurring version must reject validity")
	}
	zero := base
	zero.PriceCents = 0
	if ValidMembershipVersion(zero) {
		t.Fatal("zero price must be rejected")
	}
	duplicate := base
	duplicate.Entitlements = append(duplicate.Entitlements, MembershipPlanEntitlement{ModalityID: "m1", AccessType: MembershipAccessOpen, Position: 1})
	if ValidMembershipVersion(duplicate) {
		t.Fatal("duplicate target must be rejected")
	}
	invalidCurrency := base
	invalidCurrency.Currency = "USD"
	if ValidMembershipVersion(invalidCurrency) {
		t.Fatal("non-BRL currency must be rejected")
	}
	for _, test := range []struct {
		count int
		unit  MembershipValidityUnit
		valid bool
	}{
		{1, MembershipValidityDay, true}, {3650, MembershipValidityDay, true}, {3651, MembershipValidityDay, false},
		{520, MembershipValidityWeek, true}, {521, MembershipValidityWeek, false},
		{120, MembershipValidityMonth, true}, {121, MembershipValidityMonth, false},
		{10, MembershipValidityYear, true}, {11, MembershipValidityYear, false},
	} {
		candidate := base
		candidate.BillingCycle, candidate.ValidityCount, candidate.ValidityUnit = MembershipBillingOneTime, &test.count, &test.unit
		if actual := ValidMembershipVersion(candidate); actual != test.valid {
			t.Fatalf("validity %d %s expected %v, got %v", test.count, test.unit, test.valid, actual)
		}
	}
}

func TestMembershipPlanSummaryIsDeterministic(t *testing.T) {
	quantity, daily := 2, 1
	period := MembershipPeriodWeek
	version := MembershipPlanVersion{PriceCents: 26000, BillingCycle: MembershipBillingMonthly, Entitlements: []MembershipPlanEntitlement{{TargetName: "Pilates", ModalityID: "m1", AccessType: MembershipAccessPerPeriod, Quantity: &quantity, PeriodUnit: &period, DailyLimit: &daily}}}
	if got := MembershipPlanSummary(version); got != "Pilates · 2 sessões por semana · R$ 260/mês" {
		t.Fatalf("unexpected summary: %q", got)
	}
	validityCount, validityUnit := 90, MembershipValidityDay
	pack := MembershipPlanVersion{PriceCents: 50000, BillingCycle: MembershipBillingOneTime, ValidityCount: &validityCount, ValidityUnit: &validityUnit, Entitlements: []MembershipPlanEntitlement{{TargetName: "Pilates", ModalityID: "m1", AccessType: MembershipAccessClassPack, Quantity: func() *int { v := 10; return &v }()}}}
	if got := MembershipPlanSummary(pack); got != "Pilates · pacote de 10 aulas por 90 dias · R$ 500 pagamento único" {
		t.Fatalf("unexpected pack summary: %q", got)
	}
}

func TestMembershipAddonValidationAndSummary(t *testing.T) {
	name, ok := NormalizeMembershipAddonName("  Toalha   Extra  ")
	if !ok || name != "Toalha Extra" {
		t.Fatalf("unexpected addon name: %q %v", name, ok)
	}

	validAddon := MembershipPlanVersionAddon{
		Name:            "Recovery Plus",
		PriceDeltaCents: 5000,
		Entitlements: []MembershipPlanAddonEntitlement{
			{ServiceOfferingID: "s1", TargetName: "Recovery", AccessType: MembershipAccessOpen},
		},
	}
	if !ValidMembershipAddon(validAddon) {
		t.Fatal("expected valid addon")
	}
	if got := MembershipAddonSummary(validAddon); got != "Recovery · acesso livre" {
		t.Fatalf("unexpected addon summary: %q", got)
	}

	freeAddon := validAddon
	freeAddon.PriceDeltaCents = 0
	if !ValidMembershipAddon(freeAddon) {
		t.Fatal("expected 0 delta price addon to be valid")
	}

	negativePriceAddon := validAddon
	negativePriceAddon.PriceDeltaCents = -100
	if ValidMembershipAddon(negativePriceAddon) {
		t.Fatal("negative price delta addon must be rejected")
	}

	emptyEntitlementsAddon := validAddon
	emptyEntitlementsAddon.Entitlements = nil
	if ValidMembershipAddon(emptyEntitlementsAddon) {
		t.Fatal("addon with zero entitlements must be rejected")
	}

	version := MembershipPlanVersion{
		Name:         "CrossFit Básico",
		PriceCents:   30000,
		Currency:     "BRL",
		BillingCycle: MembershipBillingMonthly,
		Entitlements: []MembershipPlanEntitlement{{ModalityID: "m1", AccessType: MembershipAccessOpen}},
		Addons: []MembershipPlanVersionAddon{
			validAddon,
		},
	}
	if !ValidMembershipVersion(version) {
		t.Fatal("version with valid addon must be valid")
	}

	duplicateAddonVersion := version
	duplicateAddonVersion.Addons = append(duplicateAddonVersion.Addons, MembershipPlanVersionAddon{
		Name:            "recovery plus",
		PriceDeltaCents: 3000,
		Entitlements:    []MembershipPlanAddonEntitlement{{ServiceOfferingID: "s2", AccessType: MembershipAccessOpen}},
	})
	if ValidMembershipVersion(duplicateAddonVersion) {
		t.Fatal("version with duplicate addon name must be rejected")
	}
}
