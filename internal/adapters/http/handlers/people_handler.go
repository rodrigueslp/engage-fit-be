package handlers

import (
	"errors"
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
	return dto.PersonDuplicateMatchResponse{PersonID: string(match.PersonID), Name: match.Name, PhoneMasked: match.PhoneE164, RelationshipStatus: string(match.RelationshipStatus)}
}

func respondPeopleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, people.ErrInvalidInput):
		respondPublicError(c, http.StatusBadRequest, "invalid_people_input", "dados de cadastro inválidos")
	case errors.Is(err, people.ErrCPFInvalid):
		respondPublicError(c, http.StatusBadRequest, "person_cpf_invalid", "CPF inválido")
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
	default:
		respondError(c, err)
	}
}
