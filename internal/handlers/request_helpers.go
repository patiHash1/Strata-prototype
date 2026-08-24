package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/utils"
)

// requireUserID extracts and parses the authenticated user ID from the request.
// On failure it writes the error response and returns uuid.Nil, false.
func requireUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		utils.WriteErr(w, http.StatusUnauthorized, "invalid user identity")
		return uuid.Nil, false
	}
	return userID, true
}

// requireOrgID extracts and parses the authenticated org ID from the request.
// On failure it writes the error response and returns uuid.Nil, false.
func requireOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return uuid.Nil, false
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusUnauthorized, "invalid identity")
		return uuid.Nil, false
	}
	return orgID, true
}

// requireBothIDs extracts and parses both the authenticated user and org IDs.
// On failure it writes the error response and returns (nil, nil, false).
func requireBothIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	claims := utils.GetClaims(r)
	if claims == nil {
		utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
		return uuid.Nil, uuid.Nil, false
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		utils.WriteErr(w, http.StatusUnauthorized, "invalid user identity")
		return uuid.Nil, uuid.Nil, false
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		utils.WriteErr(w, http.StatusUnauthorized, "invalid org identity")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, orgID, true
}

// requirePathUUID parses a UUID-valued path parameter. On failure it writes a
// 400 and returns uuid.Nil, false.
func requirePathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

// requirePathID parses an int64-valued path parameter. On failure it writes a
// 400 and returns 0, false.
func requirePathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}

// decodeJSON decodes the request body into dst. On failure it writes a 400 and
// returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// serviceError maps a sentinel error to an HTTP status and response message.
// An empty message means "use err.Error()". It is the single home for the
// sentinel→HTTP vocabulary; handlers call writeServiceErr instead of hand-
// rolling errors.Is ladders.
var serviceError = map[error]struct {
	status  int
	message string
}{
	// Account
	services.ErrEmailAlreadyExists: {http.StatusConflict, ""},
	// Org
	services.ErrOrgAlreadyExists:         {http.StatusConflict, ""},
	services.ErrMemberNotFound:           {http.StatusNotFound, ""},
	services.ErrMemberNotInOrg:           {http.StatusNotFound, "member not found in this organization"},
	services.ErrSelfChange:               {http.StatusForbidden, ""},
	services.ErrMemberAlreadyDeactivated: {http.StatusBadRequest, ""},
	// CRM
	services.ErrQuoteNotFound:    {http.StatusNotFound, "quote not found"},
	services.ErrQuoteNotInOrg:    {http.StatusNotFound, "quote not found in this organization"},
	services.ErrContactNotFound:  {http.StatusNotFound, "contact not found"},
	services.ErrContactNotInOrg:  {http.StatusNotFound, "contact not found in this organization"},
	services.ErrCampaignNotFound: {http.StatusNotFound, "campaign not found"},
	services.ErrCampaignNotInOrg: {http.StatusNotFound, "campaign not found in this organization"},
	// Accounting
	services.ErrNoJournalItems:  {http.StatusBadRequest, ""},
	services.ErrUnbalancedEntry: {http.StatusBadRequest, ""},
	services.ErrAccountNotFound: {http.StatusNotFound, ""},
	services.ErrAccountNotInOrg: {http.StatusNotFound, ""},
	services.ErrAssetNotFound:   {http.StatusNotFound, ""},
	// HR
	services.ErrEmployeeNotFound: {http.StatusNotFound, "no employee record found for this user. Contact your HR administrator."},
	// SupplyChain
	services.ErrVehicleNotFound:     {http.StatusNotFound, "vehicle not found"},
	services.ErrNoShipmentsProvided: {http.StatusBadRequest, ""},
	services.ErrNoVehiclesProvided:  {http.StatusBadRequest, ""},
	services.ErrShipmentsNotFound:   {http.StatusNotFound, ""},
	services.ErrVehiclesNotFound:    {http.StatusNotFound, ""},
}

// writeServiceErr maps a service error to an HTTP response. If the error is a
// recognized sentinel it writes the mapped status/message and returns true;
// otherwise it writes a 500 with fallbackMsg and returns true. Callers use it
// as: if writeServiceErr(w, err, "could not ...") { return }.
func writeServiceErr(w http.ResponseWriter, err error, fallbackMsg string) bool {
	if m, ok := serviceError[err]; ok {
		msg := m.message
		if msg == "" {
			msg = err.Error()
		}
		utils.WriteErr(w, m.status, msg)
		return true
	}
	utils.WriteErr(w, http.StatusInternalServerError, fallbackMsg)
	return true
}
