package dto

type UnitResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Timezone  string `json:"timezone"`
	Status    string `json:"status"`
	IsCurrent bool   `json:"is_current"`
}

type CreateUnitRequest struct {
	Name     string `json:"name" binding:"required"`
	Timezone string `json:"timezone"`
}

type UpdateUnitRequest struct {
	Name     *string `json:"name"`
	Timezone *string `json:"timezone"`
	Status   *string `json:"status"`
	Reason   *string `json:"reason"`
}

type ReplaceMemberUnitAccessRequest struct {
	UnitIDs []string `json:"unit_ids" binding:"required"`
}

type MemberUnitAccessResponse struct {
	UnitIDs []string `json:"unit_ids"`
}
