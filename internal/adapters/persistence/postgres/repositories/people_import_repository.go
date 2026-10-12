package repositories

import (
	"context"
	"strings"

	"boxengage/backend/internal/adapters/persistence/postgres/models"
	"boxengage/backend/internal/app/people"
	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EnsureImportedStudent is the people-owned compatibility bridge for imports.
// It participates in the import transaction supplied through ctx and locks the
// local student so concurrent reconciliation cannot create duplicate people.
func (r PeopleGormRepository) EnsureImportedStudent(ctx context.Context, unitID, studentID domain.ID) (domain.ID, error) {
	var personID domain.ID
	err := databaseForContext(r.db, ctx).Transaction(func(tx *gorm.DB) error {
		var student models.StudentModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND box_id = ?", stringID(studentID), stringID(unitID)).Take(&student).Error; err != nil {
			return err
		}
		// An import must never restore an anonymized person's data or access.
		if student.AnonymizedAt != nil {
			if student.PersonID != nil {
				personID = domainID(*student.PersonID)
			}
			return nil
		}
		var unit struct{ OrganizationID string }
		if err := tx.Table("boxes").Select("organization_id").Where("id = ?", stringID(unitID)).Take(&unit).Error; err != nil {
			return err
		}
		if student.PersonID == nil {
			name := []rune(strings.Join(strings.Fields(student.Name), " "))
			if len(name) > 160 {
				name = name[:160]
			}
			if len(name) < 2 {
				name = []rune("Aluno")
			}
			person := models.PersonModel{
				ID: student.ID, OrganizationID: unit.OrganizationID, Name: string(name),
				ProfileStatus: "active", Revision: 1, CreatedAt: student.CreatedAt, UpdatedAt: student.UpdatedAt,
			}
			if phone, valid := people.NormalizeBrazilianPhone(student.Phone); valid {
				person.PhoneE164 = stringPointer(phone)
			}
			if email, valid := people.NormalizeEmail(student.Email); valid {
				person.Email = stringPointer(email)
			}
			if err := tx.Create(&person).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.StudentModel{}).Where("id = ? AND box_id = ?", student.ID, student.BoxID).
				Update("person_id", person.ID).Error; err != nil {
				return err
			}
			student.PersonID = &person.ID
		}
		var person models.PersonModel
		if err := tx.Where("id = ? AND organization_id = ? AND merged_into_person_id IS NULL", *student.PersonID, unit.OrganizationID).
			Take(&person).Error; err != nil {
			return err
		}
		personID = domainID(person.ID)
		if person.ProfileStatus != "active" {
			return nil
		}
		startedAt := student.CreatedAt
		if student.MembershipStartedAt != nil {
			startedAt = *student.MembershipStartedAt
		}
		relationship := models.PersonUnitRelationshipModel{
			ID: uuid.NewString(), OrganizationID: unit.OrganizationID, PersonID: person.ID,
			UnitID: student.BoxID, RelationshipStatus: "active", StartedAt: startedAt,
			CreatedAt: student.CreatedAt, UpdatedAt: student.UpdatedAt,
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "person_id"}, {Name: "unit_id"}}, DoNothing: true,
		}).Create(&relationship).Error
	})
	return personID, err
}

var _ portrepo.ImportedStudentPersonLinker = PeopleGormRepository{}
