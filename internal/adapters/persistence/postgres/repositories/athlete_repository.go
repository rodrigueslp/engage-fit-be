package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/domain"
	ports "boxengage/backend/internal/ports/repositories"
)

type AthleteGormRepository struct {
	db *gorm.DB
}

func NewAthleteGormRepository(db *gorm.DB) AthleteGormRepository {
	return AthleteGormRepository{db: db}
}

type athleteAccountModel struct {
	ID           string `gorm:"primaryKey"`
	Name         string
	Email        string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (athleteAccountModel) TableName() string { return "athlete_accounts" }

type athleteMembershipModel struct {
	ID               string `gorm:"primaryKey"`
	AthleteAccountID string
	BoxID            string
	Status           string
	JoinedAt         time.Time
	LeftAt           *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (athleteMembershipModel) TableName() string { return "athlete_box_memberships" }

type athleteStudentLinkModel struct {
	ID           string `gorm:"primaryKey"`
	MembershipID string
	StudentID    string
	LinkMethod   string
	LinkedAt     time.Time
	CreatedAt    time.Time
}

func (athleteStudentLinkModel) TableName() string { return "athlete_student_links" }

type athleteInvitationModel struct {
	ID              string `gorm:"primaryKey"`
	BoxID           string
	StudentID       string
	TokenHash       string
	CreatedByUserID string
	ExpiresAt       time.Time
	ClaimedAt       *time.Time
	CreatedAt       time.Time
}

func (athleteInvitationModel) TableName() string { return "athlete_invitations" }

type athleteSessionModel struct {
	ID               string `gorm:"primaryKey"`
	AthleteAccountID string
	TokenHash        string
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	LastSeenAt       time.Time
}

func (athleteSessionModel) TableName() string { return "athlete_sessions" }

func (r AthleteGormRepository) SaveInvitation(ctx context.Context, invitation *domain.AthleteInvitation) error {
	if err := ensureID(&invitation.ID); err != nil {
		return err
	}
	model := athleteInvitationModel{ID: string(invitation.ID), BoxID: string(invitation.BoxID), StudentID: string(invitation.StudentID), TokenHash: invitation.TokenHash, CreatedByUserID: string(invitation.CreatedByUserID), ExpiresAt: invitation.ExpiresAt, CreatedAt: invitation.CreatedAt}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r AthleteGormRepository) FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*domain.AthleteInvitation, error) {
	type invitationRow struct {
		athleteInvitationModel
		BoxName     string
		StudentName string
	}
	var row invitationRow
	err := r.db.WithContext(ctx).Table("athlete_invitations ai").
		Select("ai.*, boxes.name AS box_name, students.name AS student_name").
		Joins("JOIN boxes ON boxes.id = ai.box_id").
		Joins("JOIN students ON students.id = ai.student_id AND students.box_id = ai.box_id").
		Where("ai.token_hash = ?", tokenHash).Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &domain.AthleteInvitation{ID: domain.ID(row.ID), BoxID: domain.ID(row.BoxID), BoxName: row.BoxName, StudentID: domain.ID(row.StudentID), StudentName: row.StudentName, TokenHash: row.TokenHash, CreatedByUserID: domain.ID(row.CreatedByUserID), ExpiresAt: row.ExpiresAt, ClaimedAt: row.ClaimedAt, CreatedAt: row.CreatedAt}, nil
}

func (r AthleteGormRepository) FindAccountByEmail(ctx context.Context, email string) (*domain.AthleteAccount, error) {
	var model athleteAccountModel
	if err := r.db.WithContext(ctx).Where("LOWER(email) = LOWER(?)", email).Take(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ports.ErrAthleteAccountNotFound
		}
		return nil, err
	}
	return athleteAccountToDomain(model), nil
}

func (r AthleteGormRepository) ClaimInvitation(ctx context.Context, invitationID domain.ID, account *domain.AthleteAccount, existingAthleteID domain.ID, now time.Time) (*domain.AthleteMembership, error) {
	var result domain.AthleteMembership
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invitation athleteInvitationModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", string(invitationID)).Take(&invitation).Error; err != nil {
			return ports.ErrAthleteInvitationUnavailable
		}
		if invitation.ClaimedAt != nil || !invitation.ExpiresAt.After(now) {
			return ports.ErrAthleteInvitationUnavailable
		}

		var student models.StudentModel
		if err := tx.Where("id = ? AND box_id = ? AND anonymized_at IS NULL", invitation.StudentID, invitation.BoxID).Take(&student).Error; err != nil {
			return ports.ErrAthleteInvitationUnavailable
		}

		athleteID := existingAthleteID
		if account != nil {
			if err := ensureID(&account.ID); err != nil {
				return err
			}
			athleteID = account.ID
			model := athleteAccountModel{ID: string(account.ID), Name: account.Name, Email: account.Email, PasswordHash: account.PasswordHash, Status: account.Status, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt}
			if err := tx.Create(&model).Error; err != nil {
				return ports.ErrAthleteIdentityConflict
			}
		}
		if athleteID == "" {
			return ports.ErrAthleteIdentityConflict
		}

		var membership athleteMembershipModel
		err := tx.Where("athlete_account_id = ? AND box_id = ?", string(athleteID), invitation.BoxID).Take(&membership).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			membership = athleteMembershipModel{ID: uuid.NewString(), AthleteAccountID: string(athleteID), BoxID: invitation.BoxID, Status: "active", JoinedAt: now, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if membership.Status != "active" {
			if err := tx.Model(&membership).Updates(map[string]any{"status": "active", "left_at": nil, "updated_at": now}).Error; err != nil {
				return err
			}
		}

		link := athleteStudentLinkModel{ID: uuid.NewString(), MembershipID: membership.ID, StudentID: invitation.StudentID, LinkMethod: "individual_invite", LinkedAt: now, CreatedAt: now}
		if err := tx.Create(&link).Error; err != nil {
			return ports.ErrAthleteIdentityConflict
		}
		if err := tx.Model(&invitation).Update("claimed_at", now).Error; err != nil {
			return err
		}
		result = domain.AthleteMembership{ID: domain.ID(membership.ID), AthleteID: athleteID, BoxID: domain.ID(membership.BoxID), Status: "active", JoinedAt: membership.JoinedAt}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r AthleteGormRepository) SaveSession(ctx context.Context, session *domain.AthleteSession) error {
	if err := ensureID(&session.ID); err != nil {
		return err
	}
	model := athleteSessionModel{ID: string(session.ID), AthleteAccountID: string(session.AthleteID), TokenHash: session.TokenHash, ExpiresAt: session.ExpiresAt, CreatedAt: session.CreatedAt, LastSeenAt: session.CreatedAt}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r AthleteGormRepository) FindContextBySessionHash(ctx context.Context, tokenHash string, now time.Time) (*domain.AthleteContext, error) {
	var session athleteSessionModel
	if err := r.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, now).Take(&session).Error; err != nil {
		return nil, err
	}
	var account athleteAccountModel
	if err := r.db.WithContext(ctx).Where("id = ? AND status = 'active'", session.AthleteAccountID).Take(&account).Error; err != nil {
		return nil, err
	}
	type membershipRow struct {
		athleteMembershipModel
		BoxName string
	}
	var rows []membershipRow
	if err := r.db.WithContext(ctx).Table("athlete_box_memberships abm").Select("abm.*, boxes.name AS box_name").Joins("JOIN boxes ON boxes.id = abm.box_id AND boxes.status = 'active'").Where("abm.athlete_account_id = ? AND abm.status = 'active'", session.AthleteAccountID).Order("abm.joined_at ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	memberships := make([]domain.AthleteMembership, 0, len(rows))
	for _, row := range rows {
		memberships = append(memberships, domain.AthleteMembership{ID: domain.ID(row.ID), AthleteID: domain.ID(row.AthleteAccountID), BoxID: domain.ID(row.BoxID), BoxName: row.BoxName, Status: row.Status, JoinedAt: row.JoinedAt})
	}
	if len(memberships) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	_ = r.db.WithContext(ctx).Model(&session).Update("last_seen_at", now).Error
	return &domain.AthleteContext{Account: *athleteAccountToDomain(account), Memberships: memberships}, nil
}

func (r AthleteGormRepository) RevokeSession(ctx context.Context, tokenHash string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&athleteSessionModel{}).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", now).Error
}

func (r AthleteGormRepository) ListPublishedWorkouts(ctx context.Context, athleteID domain.ID) ([]domain.AthleteWorkout, error) {
	type boxRow struct {
		BoxID   string
		BoxName string
	}
	var boxes []boxRow
	if err := r.db.WithContext(ctx).Table("athlete_box_memberships abm").Select("abm.box_id, boxes.name AS box_name").Joins("JOIN boxes ON boxes.id = abm.box_id AND boxes.status = 'active'").Where("abm.athlete_account_id = ? AND abm.status = 'active'", string(athleteID)).Scan(&boxes).Error; err != nil {
		return nil, err
	}
	boxIDs := make([]string, 0, len(boxes))
	boxNames := make(map[string]string, len(boxes))
	for _, box := range boxes {
		boxIDs = append(boxIDs, box.BoxID)
		boxNames[box.BoxID] = box.BoxName
	}
	if len(boxIDs) == 0 {
		return []domain.AthleteWorkout{}, nil
	}
	var workoutModels []models.WorkoutModel
	if err := r.db.WithContext(ctx).Where("box_id IN ? AND status = ?", boxIDs, string(domain.WorkoutStatusPublished)).Order("workout_date DESC, created_at DESC").Limit(90).Find(&workoutModels).Error; err != nil {
		return nil, err
	}
	result := make([]domain.AthleteWorkout, 0, len(workoutModels))
	for _, model := range workoutModels {
		result = append(result, domain.AthleteWorkout{Workout: workoutToDomain(model), BoxName: boxNames[model.BoxID]})
	}
	return result, nil
}

func athleteAccountToDomain(model athleteAccountModel) *domain.AthleteAccount {
	return &domain.AthleteAccount{ID: domain.ID(model.ID), Name: model.Name, Email: model.Email, PasswordHash: model.PasswordHash, Status: model.Status, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}
