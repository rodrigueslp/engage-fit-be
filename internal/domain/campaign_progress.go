package domain

import "time"

type CampaignProgress struct {
	ID                 ID
	CampaignID         ID
	StudentID          ID
	CurrentCheckins    int
	TargetCheckins     int
	ProgressPercentage float64
	Achieved           bool
	NearGoal           bool
	UpdatedAt          time.Time
}

func NewCampaignProgress(campaignID, studentID ID, currentCheckins, targetCheckins int) CampaignProgress {
	progress := CampaignProgress{
		CampaignID:      campaignID,
		StudentID:       studentID,
		CurrentCheckins: currentCheckins,
		TargetCheckins:  targetCheckins,
		UpdatedAt:       time.Now(),
	}
	progress.Recalculate()
	return progress
}

func BuildCampaignProgress(campaignID ID, students []Student, checkins []Checkin, goals []CampaignGoal) []CampaignProgress {
	targetsBySource := make(map[Source]int, len(goals))
	for _, goal := range goals {
		targetsBySource[goal.Source] = goal.TargetCheckins
	}

	checkinsByStudent := make(map[ID]int)
	for _, checkin := range checkins {
		checkinsByStudent[checkin.StudentID]++
	}

	progress := make([]CampaignProgress, 0, len(checkinsByStudent))
	for _, student := range students {
		currentCheckins := checkinsByStudent[student.ID]
		if currentCheckins == 0 {
			continue
		}
		target, ok := targetsBySource[student.Source]
		if !ok {
			continue
		}
		progress = append(progress, NewCampaignProgress(campaignID, student.ID, currentCheckins, target))
	}
	return progress
}

func (p *CampaignProgress) Recalculate() {
	if p.TargetCheckins <= 0 {
		p.ProgressPercentage = 0
		p.Achieved = false
		p.NearGoal = false
		return
	}

	p.ProgressPercentage = float64(p.CurrentCheckins) / float64(p.TargetCheckins) * 100
	p.Achieved = p.CurrentCheckins >= p.TargetCheckins
	p.NearGoal = p.ProgressPercentage >= 80
}
