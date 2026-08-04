package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/hr/attendance/clock-in ----

type clockInRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ClockInResponse represents the response from a geofenced clock-in.
type ClockInResponse struct {
	AttendanceLogID  string `json:"attendance_log_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ClockIn          string `json:"clock_in" example:"2025-01-15T09:00:00Z"`
	IsWithinGeofence bool   `json:"is_within_geofence" example:"true"`
}

// clockInHandler records an employee attendance clock-in with geofence validation.
//
//	@Summary		Geofenced clock-in
//	@Description	Records an employee attendance clock-in event with GPS coordinates. Validates whether the location is within a configured geofence. The employee is identified from the JWT token's user identity. Requires `hr.attendance.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	clockInRequest	true	"Clock-in payload with GPS coordinates"
//	@Success		200	{object}	ClockInResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/hr/attendance/clock-in [post]
func (a *App) clockInHandler(w http.ResponseWriter, r *http.Request) {
	var req clockInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Latitude < -90 || req.Latitude > 90 {
		utils.WriteErr(w, http.StatusBadRequest, "latitude must be between -90 and 90")
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		utils.WriteErr(w, http.StatusBadRequest, "longitude must be between -180 and 180")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid user in token")
		return
	}

	result, err := a.HR.ClockIn(r.Context(), orgID, userID, req.Latitude, req.Longitude)
	if err != nil {
		if err == services.ErrEmployeeNotFound {
			utils.WriteErr(w, http.StatusNotFound, "no employee record found for this user. Contact your HR administrator.")
			return
		}
		utils.WriteErr(w, http.StatusInternalServerError, "could not record clock-in")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"attendance_log_id":  result.AttendanceLogID.String(),
		"clock_in":           result.ClockIn.Format(time.RFC3339),
		"is_within_geofence": result.IsWithinGeofence,
	})
}

// ---- POST /api/v1/hr/ats/parse-resume ----

const maxResumeSize = 10 << 20 // 10 MB

// ParseResumeResponse represents the response from AI resume parsing.
type ParseResumeResponse struct {
	CandidateName   string   `json:"candidate_name" example:"Jane Smith"`
	Email           string   `json:"email" example:"jane.smith@email.com"`
	ExtractedSkills []string `json:"extracted_skills" example:"Python,Go,Docker,Kubernetes"`
	AIMatchScore    int      `json:"ai_match_score" example:"85"`
}

// parseResumeHandler parses a resume file (PDF/Docx) and scores match against a job description.
//
//	@Summary		Parse resume & score match
//	@Description	Accepts a resume file (PDF or Docx) via multipart form upload, extracts candidate details and skills using AI, scores the match against a specified job description, and stores the application. Requires `hr.recruitment.write` permission.
//	@Tags			HR
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			resume_file			formData	file	true	"Resume file (PDF or Docx)"
//	@Param			job_description_id	formData	string	true	"UUID of the job description to match against"
//	@Success		200	{object}	ParseResumeResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		413	{object}	utils.Envelope
//	@Router			/api/v1/hr/ats/parse-resume [post]
func (a *App) parseResumeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form (limit to 10 MB)
	if err := r.ParseMultipartForm(maxResumeSize); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "could not parse multipart form")
		return
	}

	jobDescriptionIDStr := r.FormValue("job_description_id")
	if !utils.NotBlank(jobDescriptionIDStr) {
		utils.WriteErr(w, http.StatusBadRequest, "job_description_id is required")
		return
	}

	jobDescriptionID, err := uuid.Parse(jobDescriptionIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid job_description_id")
		return
	}

	// Retrieve the uploaded file
	file, header, err := r.FormFile("resume_file")
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "resume_file is required")
		return
	}
	defer file.Close()

	// Validate file type
	fileName := header.Filename
	if !isAllowedResumeType(fileName) {
		utils.WriteErr(w, http.StatusBadRequest, "resume_file must be a PDF or Docx file")
		return
	}

	// Read the file bytes
	resumeBytes, err := io.ReadAll(io.LimitReader(file, maxResumeSize))
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not read resume file")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	result, err := a.HR.ParseResume(r.Context(), orgID, jobDescriptionID, resumeBytes, fileName)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not parse resume")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"candidate_name":   result.CandidateName,
		"email":            result.Email,
		"extracted_skills": result.ExtractedSkills,
		"ai_match_score":   result.AIMatchScore,
	})
}

// isAllowedResumeType checks whether the file extension indicates a PDF or Docx file.
func isAllowedResumeType(fileName string) bool {
	lower := ""
	for _, r := range fileName {
		if r >= 'A' && r <= 'Z' {
			lower += string(r + 32)
		} else {
			lower += string(r)
		}
	}
	return len(lower) >= 5 && (lower[len(lower)-4:] == ".pdf" || lower[len(lower)-5:] == ".docx")
}

// ---- POST /api/v1/hr/knowledge/search ----

type knowledgeSearchRequest struct {
	Query string `json:"query"`
}

// KnowledgeSearchResponse represents the RAG-powered knowledge base search result.
type KnowledgeSearchResponse struct {
	AIAnswer        string                    `json:"ai_answer" example:"Based on our knowledge base, parental leave in Australia..."  `
	SourceDocuments []services.SourceDocument `json:"source_documents"`
}

// knowledgeSearchHandler performs semantic search over the organization's knowledge base.
//
//	@Summary		RAG knowledge base semantic search
//	@Description	Performs a RAG-powered semantic search over the organization's knowledge base documents (e.g., HR policies, onboarding guides) and returns an AI-synthesized answer with source citations. Requires `knowledge.read` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	knowledgeSearchRequest	true	"Search query"
//	@Success		200	{object}	KnowledgeSearchResponse
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/knowledge/search [post]
func (a *App) knowledgeSearchHandler(w http.ResponseWriter, r *http.Request) {
	var req knowledgeSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.Query) {
		utils.WriteErr(w, http.StatusBadRequest, "query is required")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	result, err := a.HR.SearchKnowledge(r.Context(), orgID, req.Query)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not search knowledge base")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"ai_answer":        result.AIAnswer,
		"source_documents": result.SourceDocuments,
	})
}

