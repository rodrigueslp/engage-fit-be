package dto

import (
	"encoding/json"

	"boxengage/backend/internal/domain"
)

type OrganizationResponse struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Status      string                   `json:"status"`
	CurrentUnit OrganizationUnitResponse `json:"current_unit"`
}

type OrganizationUnitResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ModalityResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Color        string `json:"color"`
	Status       string `json:"status"`
	ActiveInUnit bool   `json:"active_in_unit"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type CreateModalityRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type UpdateModalityRequest struct {
	Name   *string               `json:"name"`
	Color  *string               `json:"color"`
	Status *domain.CatalogStatus `json:"status"`
}

type SetUnitModalityRequest struct {
	Active *bool `json:"active"`
}

type SpaceResponse struct {
	ID              string `json:"id"`
	UnitID          string `json:"unit_id"`
	Name            string `json:"name"`
	DefaultCapacity *int   `json:"default_capacity"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type CreateSpaceRequest struct {
	Name            string          `json:"name"`
	DefaultCapacity json.RawMessage `json:"default_capacity"`
}

type UpdateSpaceRequest struct {
	Name            *string               `json:"name"`
	DefaultCapacity json.RawMessage       `json:"default_capacity"`
	Status          *domain.CatalogStatus `json:"status"`
}
