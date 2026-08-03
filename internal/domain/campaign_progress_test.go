package domain

import "testing"

func TestBuildCampaignProgressIncludesOnlyStudentsWithCheckinsAndGoals(t *testing.T) {
	students := []Student{
		{ID: "participant", Source: SourceWellhub},
		{ID: "box-member", Source: SourceBoxMember},
		{ID: "outside-period", Source: SourceWellhub},
		{ID: "without-goal", Source: Source("other")},
	}
	checkins := []Checkin{
		{StudentID: "participant"},
		{StudentID: "box-member"},
		{StudentID: "participant"},
		{StudentID: "without-goal"},
	}
	goals := []CampaignGoal{{Source: SourceWellhub, TargetCheckins: 3}, {Source: SourceBoxMember, TargetCheckins: 1}}

	progress := BuildCampaignProgress("campaign", students, checkins, goals)

	if len(progress) != 2 {
		t.Fatalf("expected two campaign participants, got %d: %+v", len(progress), progress)
	}
	if progress[0].StudentID != "participant" || progress[0].CurrentCheckins != 2 || progress[0].TargetCheckins != 3 {
		t.Fatalf("unexpected progress: %+v", progress[0])
	}
	if progress[1].StudentID != "box-member" || !progress[1].Achieved {
		t.Fatalf("unexpected box member progress: %+v", progress[1])
	}
}
