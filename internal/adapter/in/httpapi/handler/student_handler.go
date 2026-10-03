package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	in "github.com/code-corhuila/lms-membership-api/internal/application/port/in"
	"github.com/code-corhuila/lms-membership-api/internal/application/usecase"
	"github.com/code-corhuila/lms-membership-api/internal/domain/membership"
	"github.com/code-corhuila/lms-membership-api/internal/domain/shared"
	"github.com/code-corhuila/lms-membership-api/internal/adapter/in/httpapi/middleware"
	"github.com/code-corhuila/lms-membership-api/internal/adapter/in/httpapi/response"
)

// StudentHandler implements the /students endpoints (HU-02, HU-03). It depends on
// application/port/in interfaces, not the concrete usecase.X structs, so a use
// case's implementation can change without this file changing
// (rules/2-anexos/C-api-hexagonal.md).
type StudentHandler struct {
	createStudent     in.CreateStudentUseCase
	getStudent        in.GetStudentUseCase
	updateStudent     in.UpdateStudentUseCase
	deactivateStudent in.DeactivateStudentUseCase
	searchStudents    in.SearchStudentsUseCase
	suspendStudent    in.SuspendStudentUseCase
}

func NewStudentHandler(
	createStudent in.CreateStudentUseCase,
	getStudent in.GetStudentUseCase,
	updateStudent in.UpdateStudentUseCase,
	deactivateStudent in.DeactivateStudentUseCase,
	searchStudents in.SearchStudentsUseCase,
	suspendStudent in.SuspendStudentUseCase,
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

// clampPage and clampLimit mirror the same bounds usecase.SearchStudents applies
// server-side, so List can echo the values actually served — see
// rules/2-anexos/C-api-hexagonal.md, "Listados": meta must repeat page and limit,
// not just the total.
func clampPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func clampLimit(limit int) int {
	if limit < 1 || limit > 100 {
		return 20
	}
	return limit
}

// Create — POST /students (HU-02, FR-003, FR-004). Idempotent by the
// Idempotency-Key header (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.8): a
// retried request with the same key returns the original student and 200, not
// a second student and 201.
func (h *StudentHandler) Create(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req createStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", traceID)
		return
	}

	student, replayed, err := h.createStudent.Execute(r.Context(), req.FullName, req.DocumentID, req.Email, req.Phone, idempotencyKey)
	switch {
	case errors.Is(err, usecase.ErrDocumentIDAlreadyExists):
		response.Error(w, http.StatusConflict, "DOCUMENT_ID_ALREADY_EXISTS", "This document ID is already registered", traceID)
		return
	case errors.Is(err, shared.ErrInvalidEmail):
		response.ValidationError(w, traceID, response.FieldDetail{Field: "email", Message: "must be a valid email address"})
		return
	case err != nil:
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), traceID)
		return
	}

	if replayed {
		response.JSON(w, http.StatusOK, toStudentResponse(student))
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/students/%s", student.ID))
	response.JSON(w, http.StatusCreated, toStudentResponse(student))
}

// Get — GET /students/{id}. Used by circulation-service to check a student's
// suspension status before registering a loan (HU-06) — Membership can't be
// queried by SQL join anymore now that Loan lives in a different database.
func (h *StudentHandler) Get(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	student, err := h.getStudent.Execute(r.Context(), id)
	switch {
	case errors.Is(err, membership.ErrStudentNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Student not found", traceID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	response.JSON(w, http.StatusOK, toStudentResponse(student))
}

// List — GET /students (HU-03, search half). meta carries the full
// {total, page, limit, totalPages} envelope, echoing the page/limit actually
// served after clamping (rules/2-anexos/C-api-hexagonal.md, "Listados").
func (h *StudentHandler) List(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())

	rawPage, _ := strconv.Atoi(r.URL.Query().Get("page"))
	rawLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")

	page := clampPage(rawPage)
	limit := clampLimit(rawLimit)

	students, total, err := h.searchStudents.Execute(r.Context(), search, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	items := make([]studentResponse, 0, len(students))
	for _, s := range students {
		items = append(items, toStudentResponse(s))
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
		},
	})
}

// Update — PATCH /students/{id} (HU-03, Scenario 1).
func (h *StudentHandler) Update(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req updateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", traceID)
		return
	}

	student, err := h.updateStudent.Execute(r.Context(), id, req.FullName, req.Email, req.Phone)
	switch {
	case errors.Is(err, membership.ErrStudentNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Student not found", traceID)
		return
	case errors.Is(err, shared.ErrInvalidEmail):
		response.ValidationError(w, traceID, response.FieldDetail{Field: "email", Message: "must be a valid email address"})
		return
	case err != nil:
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusOK, toStudentResponse(student))
}

// Deactivate — POST /students/{id}/deactivate (HU-03, Scenario 2).
func (h *StudentHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	student, err := h.deactivateStudent.Execute(r.Context(), id)
	switch {
	case errors.Is(err, membership.ErrStudentNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Student not found", traceID)
		return
	case errors.Is(err, membership.ErrStudentHasActiveLoansOrSuspension):
		// 422, not 409: the request is well-formed and doesn't collide with
		// existing state — a domain rule forbids it outright
		// (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.11 / D-G08).
		response.Error(w, http.StatusUnprocessableEntity, "STUDENT_HAS_ACTIVE_LOANS_OR_SUSPENSION",
			"This student cannot be deactivated while they have active loans or an active suspension", traceID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	response.JSON(w, http.StatusOK, toStudentResponse(student))
}

// Suspend — POST /students/{id}/suspend. Called by circulation-service when
// a return is late (HU-08, INV-006) — not part of the public UI-facing
// contract.
func (h *StudentHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req suspendStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", traceID)
		return
	}

	student, err := h.suspendStudent.Execute(r.Context(), id, req.Days)
	switch {
	case errors.Is(err, membership.ErrStudentNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Student not found", traceID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	response.JSON(w, http.StatusOK, toStudentResponse(student))
}
