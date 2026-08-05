package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// ---- POST /api/v1/hr/attendance/clock-out ----

// clockOutHandler records a clock-out event for the authenticated user.
//
//	@Summary		Clock out
//	@Description	Records a clock-out event for the authenticated employee, closing the open attendance log and returning hours worked. Requires `hr.attendance.clockout` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/attendance/clock-out [post]
func (a *App) clockOutHandler(w http.ResponseWriter, r *http.Request) {
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

	result, err := a.HR.ClockOut(r.Context(), orgID, userID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not clock out")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"attendance_log_id": result.AttendanceLogID.String(),
		"clock_in":          result.ClockIn,
		"clock_out":         result.ClockOut,
		"hours_worked":      result.HoursWorked,
	})
}

// ---- POST /api/v1/hr/shifts/templates ----

type createShiftTemplateRequest struct {
	Name              string  `json:"name"`
	StartTime         string  `json:"start_time"`
	EndTime           string  `json:"end_time"`
	DayOfWeek         *int    `json:"day_of_week,omitempty"`
	Department        *string `json:"department,omitempty"`
	RequiredHeadcount int     `json:"required_headcount"`
}

// createShiftTemplateHandler creates a new shift template.
//
//	@Summary		Create shift template
//	@Description	Creates a new shift template for workforce scheduling. Requires `hr.shifts.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	createShiftTemplateRequest	true	"Shift template payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/shifts/templates [post]
func (a *App) createShiftTemplateHandler(w http.ResponseWriter, r *http.Request) {
	var req createShiftTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !utils.NotBlank(req.Name) {
		utils.WriteErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if !utils.NotBlank(req.StartTime) {
		utils.WriteErr(w, http.StatusBadRequest, "start_time is required")
		return
	}
	if !utils.NotBlank(req.EndTime) {
		utils.WriteErr(w, http.StatusBadRequest, "end_time is required")
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

	template, err := a.HR.CreateShiftTemplate(r.Context(), orgID, req.Name, req.StartTime, req.EndTime, req.DayOfWeek, req.Department, req.RequiredHeadcount)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create shift template")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"shift_template_id":  template.ID.String(),
		"name":               template.Name,
		"start_time":         template.StartTime,
		"end_time":           template.EndTime,
		"required_headcount": template.RequiredHeadcount,
	})
}

// ---- POST /api/v1/hr/shifts/assignments ----

type assignShiftRequest struct {
	EmployeeID      string `json:"employee_id"`
	ShiftTemplateID string `json:"shift_template_id"`
	ShiftDate       string `json:"shift_date"`
}

// assignShiftHandler assigns an employee to a shift on a specific date.
//
//	@Summary		Assign shift
//	@Description	Assigns an employee to a shift template on a specific date. Requires `hr.shifts.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	assignShiftRequest	true	"Shift assignment payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/shifts/assignments [post]
func (a *App) assignShiftHandler(w http.ResponseWriter, r *http.Request) {
	var req assignShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	employeeID, err := uuid.Parse(req.EmployeeID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid employee_id")
		return
	}
	shiftTemplateID, err := uuid.Parse(req.ShiftTemplateID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid shift_template_id")
		return
	}
	if !utils.NotBlank(req.ShiftDate) {
		utils.WriteErr(w, http.StatusBadRequest, "shift_date is required")
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

	assignment, err := a.HR.AssignShift(r.Context(), orgID, employeeID, shiftTemplateID, req.ShiftDate)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not assign shift")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"shift_assignment_id": assignment.ID.String(),
		"employee_id":         assignment.EmployeeID.String(),
		"shift_date":          assignment.ShiftDate,
		"status":              assignment.Status,
	})
}

// ---- GET /api/v1/hr/shifts/predictions ----

// predictShiftNeedsHandler returns AI-simulated staffing predictions for a date range.
//
//	@Summary		Predict shift needs
//	@Description	Returns AI-simulated staffing predictions for a date range based on historical attendance patterns. Requires `hr.shifts.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			department	query	string	false	"Department filter"
//	@Param			from_date	query	string	true	"Start date (YYYY-MM-DD)"
//	@Param			to_date		query	string	true	"End date (YYYY-MM-DD)"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/shifts/predictions [get]
func (a *App) predictShiftNeedsHandler(w http.ResponseWriter, r *http.Request) {
	fromDate := r.URL.Query().Get("from_date")
	toDate := r.URL.Query().Get("to_date")
	department := r.URL.Query().Get("department")

	if !utils.NotBlank(fromDate) {
		utils.WriteErr(w, http.StatusBadRequest, "from_date is required")
		return
	}
	if !utils.NotBlank(toDate) {
		utils.WriteErr(w, http.StatusBadRequest, "to_date is required")
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

	var dept *string
	if department != "" {
		dept = &department
	}

	predictions, err := a.HR.PredictShiftNeeds(r.Context(), orgID, dept, fromDate, toDate)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not predict shift needs")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"predictions": predictions,
	})
}

// ---- GET /api/v1/hr/shifts/schedule ----

// getEmployeeScheduleHandler returns shift assignments for an employee in a date range.
//
//	@Summary		Get employee schedule
//	@Description	Returns all shift assignments for an employee within a date range. Requires `hr.shifts.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			employee_id	query	string	true	"Employee ID"
//	@Param			from_date	query	string	true	"Start date (YYYY-MM-DD)"
//	@Param			to_date		query	string	true	"End date (YYYY-MM-DD)"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/shifts/schedule [get]
func (a *App) getEmployeeScheduleHandler(w http.ResponseWriter, r *http.Request) {
	employeeIDStr := r.URL.Query().Get("employee_id")
	fromDate := r.URL.Query().Get("from_date")
	toDate := r.URL.Query().Get("to_date")

	employeeID, err := uuid.Parse(employeeIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid employee_id")
		return
	}
	if !utils.NotBlank(fromDate) {
		utils.WriteErr(w, http.StatusBadRequest, "from_date is required")
		return
	}
	if !utils.NotBlank(toDate) {
		utils.WriteErr(w, http.StatusBadRequest, "to_date is required")
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

	schedule, err := a.HR.GetEmployeeSchedule(r.Context(), orgID, employeeID, fromDate, toDate)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not get employee schedule")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"employee_id": employeeID.String(),
		"schedule":    schedule,
	})
}

// ---- GET /api/v1/hr/payroll/runs/{run_id}/detail ----

// getPayrollRunDetailHandler returns detailed payroll run information including disbursements.
//
//	@Summary		Get payroll run detail
//	@Description	Returns a payroll run with all employee disbursements. Requires `hr.payroll.read` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			run_id	path	string	true	"Payroll run ID"
//	@Success		200	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Failure		404	{object}	utils.Envelope
//	@Router			/api/v1/hr/payroll/runs/{run_id}/detail [get]
func (a *App) getPayrollRunDetailHandler(w http.ResponseWriter, r *http.Request) {
	runIDStr := r.PathValue("run_id")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid run_id")
		return
	}

	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return
	}

	_, err = uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "invalid org in token")
		return
	}

	pr, disbursements, err := a.HR.GetPayrollRunDetail(r.Context(), runID)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not get payroll run detail")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"payroll_run":   pr,
		"disbursements": disbursements,
	})
}

// ---- POST /api/v1/hr/payroll/tax-profiles ----

type setEmployeeTaxProfileRequest struct {
	EmployeeID            string  `json:"employee_id"`
	TaxCountry            string  `json:"tax_country"`
	TaxID                 string  `json:"tax_id"`
	FilingStatus          string  `json:"filing_status"`
	Allowances            int     `json:"allowances"`
	AdditionalWithholding float64 `json:"additional_withholding"`
}

// setEmployeeTaxProfileHandler creates or updates the tax profile for an employee.
//
//	@Summary		Set employee tax profile
//	@Description	Creates or updates the tax withholding profile for an employee. Requires `hr.payroll.write` permission.
//	@Tags			HR
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	setEmployeeTaxProfileRequest	true	"Tax profile payload"
//	@Success		201	{object}	utils.Envelope
//	@Failure		400	{object}	utils.Envelope
//	@Failure		401	{object}	utils.Envelope
//	@Failure		403	{object}	utils.Envelope
//	@Router			/api/v1/hr/payroll/tax-profiles [post]
func (a *App) setEmployeeTaxProfileHandler(w http.ResponseWriter, r *http.Request) {
	var req setEmployeeTaxProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	employeeID, err := uuid.Parse(req.EmployeeID)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid employee_id")
		return
	}
	if !utils.NotBlank(req.TaxCountry) {
		utils.WriteErr(w, http.StatusBadRequest, "tax_country is required")
		return
	}
	if !utils.NotBlank(req.FilingStatus) {
		utils.WriteErr(w, http.StatusBadRequest, "filing_status is required")
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

	profile, err := a.HR.SetEmployeeTaxProfile(r.Context(), orgID, employeeID, req.TaxCountry, req.TaxID, req.FilingStatus, req.Allowances, req.AdditionalWithholding)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not set employee tax profile")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"tax_profile_id": profile.ID.String(),
		"employee_id":    profile.EmployeeID.String(),
		"tax_country":    profile.TaxCountry,
		"filing_status":  profile.FilingStatus,
	})
}
