package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"boxengage/backend/internal/adapters/http/dto"
	"boxengage/backend/internal/adapters/http/middleware"
	"boxengage/backend/internal/app/imports"
	"boxengage/backend/internal/domain"
)

type ImportsHandler struct {
	importCheckins imports.ImportCheckinsUseCase
	listImports    imports.ListImportsUseCase
	getImport      imports.GetImportUseCase
}

func NewImportsHandler(importCheckins imports.ImportCheckinsUseCase, listImports imports.ListImportsUseCase, getImport imports.GetImportUseCase) ImportsHandler {
	return ImportsHandler{importCheckins: importCheckins, listImports: listImports, getImport: getImport}
}

func (h ImportsHandler) Create(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	source := domain.Source(c.PostForm("source"))
	if source == "" {
		respondBadRequest(c)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondBadRequest(c)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		respondError(c, err)
		return
	}
	defer file.Close()

	output, err := h.importCheckins.Execute(c.Request.Context(), imports.ImportCheckinsInput{
		BoxID:    boxID,
		Source:   source,
		Filename: fileHeader.Filename,
		File:     file,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ImportResponse{
		ID:           string(output.ImportID),
		Source:       string(source),
		TotalRecords: output.TotalRecords,
		Students:     output.Students,
		Checkins:     output.Checkins,
	})
}

func (h ImportsHandler) List(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	result, err := h.listImports.Execute(c.Request.Context(), boxID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]dto.ImportResponse, 0, len(result))
	for _, item := range result {
		response = append(response, importResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h ImportsHandler) Get(c *gin.Context) {
	boxID, err := middleware.BoxID(c)
	if err != nil {
		respondUnauthorized(c)
		return
	}

	item, err := h.getImport.Execute(c.Request.Context(), boxID, domain.ID(c.Param("id")))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, importResponse(*item))
}

func importResponse(item domain.ImportHistory) dto.ImportResponse {
	return dto.ImportResponse{
		ID:           string(item.ID),
		Filename:     item.Filename,
		Source:       string(item.Source),
		TotalRecords: item.TotalRecords,
		ImportedAt:   item.ImportedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
