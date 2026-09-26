package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/code-corhuila/lms-membership-api/internal/application/usecase"
	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
	"github.com/code-corhuila/lms-membership-api/internal/infrastructure/http/middleware"
	"github.com/code-corhuila/lms-membership-api/internal/infrastructure/http/response"
)

// StudentHandler implements the /students endpoints (HU-02, HU-03).
type StudentHandler struct {
	createStudent     *usecase.CreateStudent
	getStudent        *usecase.GetStudent
	updateStudent     *usecase.UpdateStudent
	deactivateStudent *usecase.DeactivateStudent
	searchStudents    *usecase.SearchStudents
	suspendStudent    *usecase.SuspendStudent
}

func NewStudentHandler(
	createStudent *usecase.CreateStudent,
	getStudent *usecase.GetStudent,
	updateStudent *usecase.UpdateStudent,
	deactivateStudent *usecase.DeactivateStudent,
	searchStudents *usecase.SearchStudents,
	suspendStudent *usecase.SuspendStudent,
) *StudentHandler {
	return &StudentHandler{
		createStudent:     createStudent,
		getStudent:        getStudent,
		updateStudent:     updateStudent,
		deactivateStudent: deactivateStudent,
		searchStudents:    searchStudents,
		suspendStudent:    suspendStudent,
	}
}

type createStudentRequest struct {
	FullName   string `json:"fullName"`
	DocumentID string `json:"documentId"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
}

type updateStudentRequest struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type suspendStudentRequest struct {
	Days int `json:"days"`
}

type studentResponse struct {
	ID             string  `json:"id"`
	FullName       string  `json:"fullName"`
	DocumentID     string  `json:"documentId"`
	Email          string  `json:"email"`
	Phone          *string `json:"phone,omitempty"`
	SuspendedUntil *string `json:"suspendedUntil,omitempty"`
	DeactivatedAt  *string `json:"deactivatedAt,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

func toStudentResponse(s *membership.Student) studentResponse {
	resp := studentResponse{
		ID:         s.ID,
		FullName:   s.FullName,
		DocumentID: s.DocumentID,
		Email:      s.Email.String(),
		CreatedAt:  s.CreatedAt.Format(timeFormat),
		UpdatedAt:  s.UpdatedAt.Format(timeFormat),
	}
	if s.Phone != "" {
		resp.Phone = &s.Phone
	}
	if s.SuspendedUntil != nil {
		v := s.SuspendedUntil.Format(timeFormat)
		resp.SuspendedUntil = &v
	}
	if s.DeactivatedAt != nil {
		v := s.DeactivatedAt.Format(timeFormat)
		resp.DeactivatedAt = &v
	}
	return resp
}

// Create — POST /students (HU-02, FR-003, FR-004).
func (h *StudentHandler) Create(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())

	var req createStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", correlationID)
		return
	}

	student, err := h.createStudent.Execute(r.Context(), req.FullName, req.DocumentID, req.Email, req.Phone)
	switch {
	case errors.Is(err, usecase.ErrDocumentIDAlreadyExists):
		response.Error(w, http.StatusConflict, "DOCUMENT_ID_ALREADY_EXISTS", "This document ID is already registered", correlationID)
		return
	case errors.Is(err, shared.ErrInvalidEmail):
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid email format", correlationID)
		return
	case err != nil:
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), correlationID)
		return
	}

	response.JSON(w, http.StatusCreated, toStudentResponse(student))
}

// Get — GET /students/{id}. Used by circulation-service to check a student's
// suspension status before registering a loan (HU-06) — Membership can't be
// queried by SQL join anymore now that Loan lives in a different database.
func (h *StudentHandler) Get(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	student, err := h.getStudent.Execute(r.Context(), id)
	switch {
	case errors.Is(err, membership.ErrStudentNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Student not found", correlationID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", correlationID)
		return
	}

	response.JSON(w, http.StatusOK, toStudentResponse(student))
}

// List — GET /students (HU-03, search half).
func (h *StudentHandler) List(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")

	students, total, err := h.searchStudents.Execute(r.Context(), search, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", correlationID)
		return
	}

	items := make([]studentResponse, 0, len(students))
	for _, s := range students {
		items = append(items, toStudentResponse(s))
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"total": total},
	})
}

// Update — PATCH /students/{id} (HU-03, Scenario 1).
func (h *StudentHandler) Update(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req updateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", correlationID)
		return
	}

	student, err := h.updateStudent.Execute(r.Context(), id, req.FullName, req.Email, req.Phone)
	switch {
	case errors.Is(err, membership.ErrStudentNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Student not found", correlationID)
		return
	case errors.Is(err, shared.ErrInvalidEmail):
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid email format", correlationID)
		return
	case err != nil:
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), correlationID)
		return
	}

	response.JSON(w, http.StatusOK, toStudentResponse(student))
}

// Deactivate — POST /students/{id}/deactivate (HU-03, Scenario 2).
func (h *StudentHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	student, err := h.deactivateStudent.Execute(r.Context(), id)
	switch {
	case errors.Is(err, membership.ErrStudentNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Student not found", correlationID)
		return
	case errors.Is(err, membership.ErrStudentHasActiveLoansOrSuspension):
		response.Error(w, http.StatusConflict, "STUDENT_HAS_ACTIVE_LOANS_OR_SUSPENSION",
			"This student cannot be deactivated while they have active loans or an active suspension", correlationID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", correlationID)
		return
	}

	response.JSON(w, http.StatusOK, toStudentResponse(student))
}

// Suspend — POST /students/{id}/suspend. Called by circulation-service when
// a return is late (HU-08, INV-006) — not part of the public UI-facing
// contract.
func (h *StudentHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req suspendStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", correlationID)
		return
	}

	student, err := h.suspendStudent.Execute(r.Context(), id, req.Days)
	switch {
	case errors.Is(err, membership.ErrStudentNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Student not found", correlationID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", correlationID)
		return
	}

	response.JSON(w, http.StatusOK, toStudentResponse(student))
}
