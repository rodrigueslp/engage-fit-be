package domain

type CampaignGoal struct {
	ID             ID
	CampaignID     ID
	Source         Source
	TargetCheckins int
}
