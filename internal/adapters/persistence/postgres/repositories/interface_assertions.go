package repositories

import portrepo "boxengage/backend/internal/ports/repositories"

var (
	_ portrepo.BoxRepository                      = BoxGormRepository{}
	_ portrepo.CatalogRepository                  = CatalogGormRepository{}
	_ portrepo.CatalogPlanReferenceResolver       = CatalogGormRepository{}
	_ portrepo.SchedulingCatalogResolver          = CatalogGormRepository{}
	_ portrepo.MembershipRepository               = MembershipGormRepository{}
	_ portrepo.EnrollmentRepository               = EnrollmentGormRepository{}
	_ portrepo.EnrollmentRosterReader             = EnrollmentGormRepository{}
	_ portrepo.MembershipAccessRepository         = EnrollmentGormRepository{}
	_ portrepo.MembershipAccessPrerequisiteReader = EnrollmentGormRepository{}
	_ portrepo.MembershipAccessUsageReader        = MembershipAccessUsageGormAdapter{}
	_ portrepo.LegacyMembershipReader             = AttendanceGormRepository{}
	_ portrepo.ReservationEligibilityResolver     = EnrollmentGormRepository{}
	_ portrepo.EnrollmentPrivacyReader            = EnrollmentGormRepository{}
	_ portrepo.EnrollmentSubjectResolver          = PeopleGormRepository{}
	_ portrepo.EnrollmentRosterPeopleReader       = PeopleGormRepository{}
	_ portrepo.ReservationPersonResolver          = PeopleGormRepository{}
	_ portrepo.ReservationRepository              = ReservationGormRepository{}
	_ portrepo.ReservationPrivacyReader           = ReservationGormRepository{}
	_ portrepo.PeopleRepository                   = PeopleGormRepository{}
	_ portrepo.PersonIdentityLinker               = PeopleGormRepository{}
	_ portrepo.BillingRepository                  = BillingGormRepository{}
	_ portrepo.MemberBillingRepository            = MemberBillingGormRepository{}
	_ portrepo.MemberBillingPrivacyReader         = MemberBillingGormRepository{}
	_ portrepo.CommerceRepository                 = CommerceGormRepository{}
	_ portrepo.CommercePrivacyReader              = CommerceGormRepository{}
	_ portrepo.UserRepository                     = UserGormRepository{}
	_ portrepo.StudentRepository                  = StudentGormRepository{}
	_ portrepo.AthletePortalRepository            = AthleteGormRepository{}
	_ portrepo.PrivacyRepository                  = PrivacyGormRepository{}
	_ portrepo.CheckinRepository                  = CheckinGormRepository{}
	_ portrepo.ImportHistoryRepository            = ImportHistoryGormRepository{}
	_ portrepo.CampaignRepository                 = CampaignGormRepository{}
	_ portrepo.RewardRepository                   = RewardGormRepository{}
	_ portrepo.TransactionManager                 = GormTransactionManager{}
	_ portrepo.WhatsappSettingsRepository         = WhatsappSettingsGormRepository{}
	_ portrepo.MessageRepository                  = MessageGormRepository{}
	_ portrepo.MessagingGovernanceRepository      = MessagingGovernanceGormRepository{}
	_ portrepo.SchedulingRepository               = SchedulingGormRepository{}
	_ portrepo.SchedulingGridRepository           = SchedulingGormRepository{}
	_ portrepo.SchedulingUnitContextResolver      = SchedulingAccessGormAdapter{}
	_ portrepo.SchedulingAccountResolver          = SchedulingAccessGormAdapter{}
	_ portrepo.HealthFormRepository               = HealthFormGormRepository{}
	_ portrepo.ContractLifecycleRepository        = ContractLifecycleGormRepository{}
)
