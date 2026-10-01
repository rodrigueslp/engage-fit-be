package handlers

import (
	"errors"
	"io"
	"net/http"
	"time"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/people"
	"boxengage/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type PeopleHandler struct{ service *people.Service }

func NewPeopleHandler(service *people.Service) PeopleHandler { return PeopleHandler{service: service} }

func (h PeopleHandler) List(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.service.List(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), people.SearchInput{})
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h PeopleHandler) Search(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.PersonSearchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "busca inválida")
		return
	}
	result, err := h.service.List(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), request)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h PeopleHandler) Get(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	role, err := middleware.Role(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	profile, err := h.service.Get(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), domain.ID(c.Param("id")), role == domain.UserRoleOwner)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	c.JSON(http.StatusOK, personProfileResponse(profile))
}

func (h PeopleHandler) Update(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.UpdatePersonRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "dados do perfil inválidos")
		return
	}
	profile, err := h.service.Update(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), domain.ID(c.Param("id")), userID, request)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	c.JSON(http.StatusOK, personProfileResponse(profile))
}

func (h PeopleHandler) LinkUnit(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.LinkPersonUnitRequest
	if err := c.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "dados do vínculo inválidos")
		return
	}
	profile, err := h.service.LinkUnit(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), domain.ID(c.Param("id")), userID, request)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, personProfileResponse(profile))
}

func (h PeopleHandler) LinkExternalIdentity(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.LinkExternalIdentityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "identidade externa inválida")
		return
	}
	profile, err := h.service.LinkExternalIdentity(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), domain.ID(c.Param("id")), userID, request)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, personProfileResponse(profile))
}

func (h PeopleHandler) Merge(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.MergePeopleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "consolidação inválida")
		return
	}
	profile, err := h.service.Merge(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), domain.ID(c.Param("id")), userID, request)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	c.JSON(http.StatusOK, personProfileResponse(profile))
}

func (h PeopleHandler) ListIdentityReviews(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	result, err := h.service.ListIdentityReviews(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")))
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h PeopleHandler) ResolveIdentityReview(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.ResolveIdentityReviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "decisão de revisão inválida")
		return
	}
	review, profile, err := h.service.ResolveIdentityReview(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), domain.ID(c.Param("review_id")), userID, request)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	response := gin.H{"review": review}
	if profile != nil {
		response["person"] = personProfileResponse(profile)
	}
	c.JSON(http.StatusOK, response)
}

func (h PeopleHandler) DuplicateCheck(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.PersonDuplicateCheckRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "dados de cadastro inválidos")
		return
	}
	result, err := h.service.DuplicateCheck(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), request)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	response := dto.PersonDuplicateCheckResponse{PhoneMatches: []dto.PersonDuplicateMatchResponse{}}
	if result.CPFMatch != nil {
		match := duplicateMatchResponse(*result.CPFMatch)
		response.CPFMatch = &match
	}
	for _, match := range result.PhoneMatches {
		response.PhoneMatches = append(response.PhoneMatches, duplicateMatchResponse(match))
	}
	c.JSON(http.StatusOK, response)
}

func (h PeopleHandler) Create(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	userID, err := middleware.UserID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}
	var request dto.CreatePersonRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "dados de cadastro inválidos")
		return
	}
	result, replayed, err := h.service.Create(c.Request.Context(), boxID, domain.ID(c.Param("unit_id")), userID, c.GetHeader("Idempotency-Key"), request)
	if err != nil {
		respondPeopleError(c, err)
		return
	}
	if replayed {
		c.Header("Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusCreated, dto.CreatePersonResponse{PersonID: string(result.PersonID), StudentID: string(result.StudentID), Name: result.Name, Phone: result.PhoneE164, Email: result.Email, CPFMasked: "***.***.***-" + result.CPFLast2, RelationshipStatus: string(result.RelationshipStatus), ContactStatus: string(result.ContactStatus), CreatedAt: result.CreatedAt.Format(time.RFC3339)})
}

func duplicateMatchResponse(match domain.PersonDuplicateMatch) dto.PersonDuplicateMatchResponse {
	status := string(match.RelationshipStatus)
	if status == "" {
		status = "not_linked"
	}
	return dto.PersonDuplicateMatchResponse{PersonID: string(match.PersonID), Name: match.Name, PhoneMasked: match.PhoneE164, RelationshipStatus: status}
}

func respondPeopleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, people.ErrInvalidInput):
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "dados de cadastro inválidos")
	case errors.Is(err, people.ErrCPFInvalid):
		respondPublicError(c, http.StatusBadRequest, "person_cpf_invalid", "CPF inválido")
	case errors.Is(err, people.ErrPhoneInvalid):
		respondPublicError(c, http.StatusBadRequest, "person_phone_invalid", "WhatsApp inválido; informe um celular brasileiro com 9 dígitos")
	case errors.Is(err, people.ErrInvalidIdempotencyKey):
		respondPublicError(c, http.StatusBadRequest, "invalid_idempotency_key", "chave de idempotência inválida")
	case errors.Is(err, people.ErrUnitNotFound):
		respondPublicError(c, http.StatusNotFound, "unit_not_found", "unidade não encontrada")
	case errors.Is(err, people.ErrCPFConflict):
		respondPublicError(c, http.StatusConflict, "person_cpf_conflict", "CPF já cadastrado")
	case errors.Is(err, people.ErrPossibleDuplicate):
		respondPublicError(c, http.StatusConflict, "person_possible_duplicate", "telefone já associado a outra pessoa")
	case errors.Is(err, people.ErrIdempotencyKeyReused):
		respondPublicError(c, http.StatusConflict, "idempotency_key_reused", "chave de idempotência reutilizada")
	case errors.Is(err, people.ErrPersonNotFound):
		respondPublicError(c, http.StatusNotFound, "person_not_found", "aluno não encontrado")
	case errors.Is(err, people.ErrPersonRevisionConflict):
		respondPublicError(c, http.StatusConflict, "person_revision_conflict", "o perfil foi alterado; atualize e tente novamente")
	case errors.Is(err, people.ErrOpenEnrollmentPhoneRequired):
		respondPublicError(c, http.StatusConflict, "open_enrollment_phone_required", "substitua o WhatsApp ou encerre a matrícula antes de remover o telefone")
	case errors.Is(err, people.ErrPersonAlreadyLinked):
		respondPublicError(c, http.StatusConflict, "person_unit_already_linked", "a pessoa já está vinculada a esta unidade")
	case errors.Is(err, people.ErrExternalIdentityConflict):
		respondPublicError(c, http.StatusConflict, "person_external_identity_conflict", "a identidade externa pertence a outra pessoa e foi enviada para revisão")
	case errors.Is(err, people.ErrIdentityReviewNotFound):
		respondPublicError(c, http.StatusNotFound, "person_identity_review_not_found", "revisão não encontrada")
	case errors.Is(err, people.ErrMergeNotAllowed):
		respondPublicError(c, http.StatusConflict, "person_merge_not_allowed", "a consolidação possui dependências incompatíveis")
	default:
		respondError(c, err)
	}
}

func personProfileResponse(profile *domain.PersonProfile) dto.PersonProfileResponse {
	response := dto.PersonProfileResponse{ID: string(profile.Person.ID), Name: profile.Person.Name, PreferredName: profile.Person.PreferredName, Phone: profile.Person.PhoneE164, Email: profile.Person.Email, CPFMasked: profile.CPFMasked, ProfileStatus: string(profile.Person.ProfileStatus), Revision: profile.Person.Revision, Relationships: profile.Relationships, SourceProfiles: profile.Profiles, ExternalIdentities: profile.Identities, AthleteAccount: profile.AthleteAccount, History: profile.History, CreatedAt: profile.Person.CreatedAt, UpdatedAt: profile.Person.UpdatedAt}
	if profile.Address != nil {
		response.Address = &dto.PersonAddressResponse{PostalCode: profile.Address.PostalCode, Street: profile.Address.Street, Number: profile.Address.Number, Complement: profile.Address.Complement, District: profile.Address.District, City: profile.Address.City, State: profile.Address.State, Country: profile.Address.Country}
	}
	return response
}
