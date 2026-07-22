package domain

import "testing"

func TestBuildCampaignProgressIncludesOnlyStudentsWithCheckinsAndGoals(t *testing.T) {
	students := []Student{
		{ID: "participant", Source: SourceWellhub},
		{ID: "outside-period", Source: SourceWellhub},
		{ID: "without-goal", Source: Source("other")},
	}
	checkins := []Checkin{
		{StudentID: "participant"},
		{StudentID: "participant"},
		{StudentID: "without-goal"},
	}
	goals := []CampaignGoal{{Source: SourceWellhub, TargetCheckins: 3}}

	progress := BuildCampaignProgress("campaign", students, checkins, goals)

	if len(progress) != 1 {
		t.Fatalf("expected one campaign participant, got %d: %+v", len(progress), progress)
	}
	if progress[0].StudentID != "participant" || progress[0].CurrentCheckins != 2 || progress[0].TargetCheckins != 3 {
		t.Fatalf("unexpected progress: %+v", progress[0])
	}
}
