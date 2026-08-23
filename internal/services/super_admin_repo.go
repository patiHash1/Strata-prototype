package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patiHash1/Strata-prototype/internal/logger"
)

// superAdminRepository handles persistence for the observability modules.
type superAdminRepository struct {
	pool *pgxpool.Pool
}

func newSuperAdminRepository(pool *pgxpool.Pool) *superAdminRepository {
	return &superAdminRepository{pool: pool}
}

func (r *superAdminRepository) UpdateMaintenanceRule(ctx context.Context, rule *MaintenanceRule) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO super_admin_maintenance_rules (scope, target_id, is_active, reason, allowed_roles, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (scope, target_id)
		DO UPDATE SET is_active = $3, reason = $4, allowed_roles = $5, updated_at = NOW()
		RETURNING id, created_at, updated_at
	`, rule.Scope, rule.TargetID, rule.IsActive, rule.Reason, rule.AllowedRoles).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *superAdminRepository) ListActiveMaintenanceRules(ctx context.Context) ([]MaintenanceRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, scope, target_id, is_active, reason, allowed_roles, created_at, updated_at
		FROM super_admin_maintenance_rules
		WHERE is_active = TRUE
		ORDER BY scope, target_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []MaintenanceRule
	for rows.Next() {
		var rule MaintenanceRule
		if err := rows.Scan(&rule.ID, &rule.Scope, &rule.TargetID, &rule.IsActive,
			&rule.Reason, &rule.AllowedRoles, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *superAdminRepository) ListAllMaintenanceRules(ctx context.Context) ([]MaintenanceRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, scope, target_id, is_active, reason, allowed_roles, created_at, updated_at
		FROM super_admin_maintenance_rules
		ORDER BY scope, target_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []MaintenanceRule
	for rows.Next() {
		var rule MaintenanceRule
		if err := rows.Scan(&rule.ID, &rule.Scope, &rule.TargetID, &rule.IsActive,
			&rule.Reason, &rule.AllowedRoles, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *superAdminRepository) DeleteMaintenanceRule(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE super_admin_maintenance_rules
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *superAdminRepository) InsertSystemError(ctx context.Context, errRec *SystemError) error {
	_, dbErr := r.pool.Exec(ctx, `
		INSERT INTO super_admin_system_errors (module, error_message, stack_trace, status_code)
		VALUES ($1, $2, $3, $4)
	`, errRec.Module, errRec.ErrorMessage, errRec.StackTrace, errRec.StatusCode)
	return dbErr
}

func (r *superAdminRepository) InsertCIHealthReport(ctx context.Context, report *CIHealthReport) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO super_admin_ci_health_reports (module, coverage_percent, linter_issues, vulnerabilities_count, commit_sha)
		VALUES ($1, $2, $3, $4, $5)
	`, report.Module, report.CoveragePercent, report.LinterIssues, report.VulnerabilitiesCount, report.CommitSHA)
	return err
}

func (r *superAdminRepository) GetLatestCIHealthByModule(ctx context.Context, module string) (*CIHealthReport, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, module, coverage_percent, linter_issues, vulnerabilities_count, commit_sha, created_at
		FROM super_admin_ci_health_reports
		WHERE module = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, module)

	var report CIHealthReport
	err := row.Scan(&report.ID, &report.Module, &report.CoveragePercent,
		&report.LinterIssues, &report.VulnerabilitiesCount, &report.CommitSHA, &report.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

func (r *superAdminRepository) GetAllLatestCIHealth(ctx context.Context) ([]CIHealthReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (module) id, module, coverage_percent, linter_issues, vulnerabilities_count, commit_sha, created_at
		FROM super_admin_ci_health_reports
		ORDER BY module, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []CIHealthReport
	for rows.Next() {
		var report CIHealthReport
		if err := rows.Scan(&report.ID, &report.Module, &report.CoveragePercent,
			&report.LinterIssues, &report.VulnerabilitiesCount, &report.CommitSHA, &report.CreatedAt); err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

// InsertSOCEvent persists a security event to the database.
func (r *superAdminRepository) InsertSOCEvent(ctx context.Context, event *SOCEvent) error {
	var metadataJSON []byte
	if event.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return err
		}
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO super_admin_soc_events (id, event_type, severity, message, ip_address, user_id, org_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, event.ID, event.Type, event.Severity, event.Message,
		event.IPAddress, event.UserID, event.OrgID, metadataJSON, event.Timestamp)
	return err
}

// PruneSOCEvents deletes SOC events older than the given duration.
func (r *superAdminRepository) PruneSOCEvents(ctx context.Context, maxAge time.Duration) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM super_admin_soc_events
		WHERE created_at < NOW() - $1::interval
	`, fmt.Sprintf("%d seconds", int(maxAge.Seconds())))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListRecentSOCEvents returns the most recent SOC events (for paginated audit views).
func (r *superAdminRepository) ListRecentSOCEvents(ctx context.Context, limit int) ([]SOCEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, event_type, severity, message, ip_address, user_id, org_id, metadata, created_at
		FROM super_admin_soc_events
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SOCEvent
	for rows.Next() {
		var e SOCEvent
		var metadataJSON []byte
		if err := rows.Scan(&e.ID, &e.Type, &e.Severity, &e.Message,
			&e.IPAddress, &e.UserID, &e.OrgID, &metadataJSON, &e.Timestamp); err != nil {
			return nil, err
		}
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				logger.Error("failed to unmarshal SOC event metadata", slog.String("error", err.Error()))
			}
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
