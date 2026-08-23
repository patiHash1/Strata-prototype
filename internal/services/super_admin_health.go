package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ModuleHealthSource provides the CI-health persistence needed by ModuleHealthSvc.
type ModuleHealthSource interface {
	GetLatestCIHealthByModule(ctx context.Context, module string) (*CIHealthReport, error)
	GetAllLatestCIHealth(ctx context.Context) ([]CIHealthReport, error)
	InsertCIHealthReport(ctx context.Context, report *CIHealthReport) error
}

// ModuleHealthSvc composes Telemetry + Maintenance into per-module health scores
// and status. It depends on their interfaces, not their concrete types.
type ModuleHealthSvc struct {
	repo ModuleHealthSource
	http HTTPMetricsSource
	main MaintenanceStatusSource
}

// MaintenanceStatusSource exposes the full maintenance rule cache for
// feature-level status lookups, including the checker interface.
type MaintenanceStatusSource interface {
	MaintenanceChecker
	CacheRules() map[string]*MaintenanceRule
}

// NewModuleHealth creates a ModuleHealthSvc module.
func NewModuleHealth(pool *pgxpool.Pool, http HTTPMetricsSource, main MaintenanceStatusSource) *ModuleHealthSvc {
	var repo ModuleHealthSource
	if pool != nil {
		repo = newSuperAdminRepository(pool)
	}
	return &ModuleHealthSvc{repo: repo, http: http, main: main}
}

// IngestCIHealth stores a CI health report.
func (m *ModuleHealthSvc) IngestCIHealth(ctx context.Context, req CIHealthIngestRequest) (*CIHealthReport, error) {
	if m.repo == nil {
		return nil, nil
	}

	report := &CIHealthReport{
		Module:               req.Module,
		CoveragePercent:      req.CoveragePercent,
		LinterIssues:         req.LinterIssues,
		VulnerabilitiesCount: req.VulnerabilitiesCount,
		CommitSHA:            req.CommitSHA,
	}

	if err := m.repo.InsertCIHealthReport(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

// GetModuleHealth computes composite health scores for all modules.
func (m *ModuleHealthSvc) GetModuleHealth(ctx context.Context) ([]ModuleHealth, error) {
	if m.repo == nil {
		return []ModuleHealth{}, nil
	}

	reports, err := m.repo.GetAllLatestCIHealth(ctx)
	if err != nil {
		return nil, err
	}

	perModule := m.http.PerModuleMetrics()

	// Build a set of known modules from CI reports.
	seen := make(map[string]bool)
	var healths []ModuleHealth

	for _, r := range reports {
		h := ModuleHealth{
			Module:          r.Module,
			CoveragePercent: r.CoveragePercent,
			LinterIssues:    r.LinterIssues,
			Vulnerabilities: r.VulnerabilitiesCount,
		}

		if pm, ok := perModule[r.Module]; ok && pm.Requests > 0 {
			h.ErrorRate5xx = float64(pm.Errors5xx) / float64(pm.Requests) * 100
		}

		// Composite health score:
		// 40% coverage, 30% linter penalty, 20% vulnerability penalty, 10% error rate penalty.
		coverageScore := r.CoveragePercent * 0.4

		linterScore := 30.0
		if r.LinterIssues > 0 {
			linterPenalty := float64(r.LinterIssues) * 0.5
			if linterPenalty > 30 {
				linterPenalty = 30
			}
			linterScore = 30 - linterPenalty
		}

		vulnScore := 20.0
		if r.VulnerabilitiesCount > 0 {
			vulnPenalty := float64(r.VulnerabilitiesCount) * 5
			if vulnPenalty > 20 {
				vulnPenalty = 20
			}
			vulnScore = 20 - vulnPenalty
		}

		errorScore := 10.0
		if h.ErrorRate5xx > 0 {
			errorPenalty := h.ErrorRate5xx * 2
			if errorPenalty > 10 {
				errorPenalty = 10
			}
			errorScore = 10 - errorPenalty
		}

		h.HealthScore = coverageScore + linterScore + vulnScore + errorScore
		healths = append(healths, h)
		seen[r.Module] = true
	}

	// Include modules that have HTTP metrics but no CI reports.
	for mod := range perModule {
		if seen[mod] {
			continue
		}
		h := ModuleHealth{
			Module: mod,
		}
		if perModule[mod].Requests > 0 {
			h.ErrorRate5xx = float64(perModule[mod].Errors5xx) / float64(perModule[mod].Requests) * 100
		}
		healths = append(healths, h)
	}

	return healths, nil
}

// GetModuleStatus returns the current health and maintenance state for a given module.
func (m *ModuleHealthSvc) GetModuleStatus(ctx context.Context, module string) (map[string]any, error) {
	status := map[string]any{
		"module":  module,
		"status":  "operational",
		"healthy": true,
	}

	// Check if the module itself is under maintenance.
	if rule, ok := m.main.IsUnderMaintenance("module", module); ok {
		status["status"] = "maintenance"
		status["healthy"] = false
		status["maintenance"] = map[string]any{
			"scope":  rule.Scope,
			"target": rule.TargetID,
			"reason": rule.Reason,
			"since":  rule.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	// Find feature-level maintenance rules targeting this module's features.
	var features []map[string]any
	for _, rule := range m.main.CacheRules() {
		if rule.Scope == "feature" && rule.IsActive {
			matched := false
			featureSlug := rule.TargetID

			// Convention 1: target_id is "module:feature-slug" or "module/feature-slug".
			if len(rule.TargetID) > len(module) && rule.TargetID[:len(module)] == module {
				rest := rule.TargetID[len(module):]
				if len(rest) >= 2 && (rest[0] == ':' || rest[0] == '/') {
					matched = true
					featureSlug = rest[1:]
				}
			}

			// Convention 3: target_id is just "feature-slug" (bare slug).
			if !matched && !strings.ContainsAny(rule.TargetID, ":/") {
				matched = true
				featureSlug = rule.TargetID
			}

			if matched {
				features = append(features, map[string]any{
					"feature": featureSlug,
					"target":  rule.TargetID,
					"reason":  rule.Reason,
					"since":   rule.CreatedAt.Format("2006-01-02T15:04:05Z"),
				})
			}
		}
	}

	if len(features) > 0 {
		status["features_under_maintenance"] = features
		if status["status"] == "operational" {
			status["status"] = "degraded"
		}
	}

	// Include CI health data if available.
	if m.repo != nil {
		report, err := m.repo.GetLatestCIHealthByModule(ctx, module)
		if err == nil && report != nil {
			ci := map[string]any{
				"coverage_percent":      report.CoveragePercent,
				"linter_issues":         report.LinterIssues,
				"vulnerabilities_count": report.VulnerabilitiesCount,
				"commit_sha":            report.CommitSHA,
			}
			status["ci_health"] = ci
		}
	}

	// Include HTTP metrics for this module.
	if pm, ok := m.http.PerModuleMetrics()[module]; ok {
		http := map[string]any{
			"total_requests": pm.Requests,
			"errors_5xx":     pm.Errors5xx,
		}
		if pm.Requests > 0 {
			http["error_rate_5xx"] = fmt.Sprintf("%.2f%%", float64(pm.Errors5xx)/float64(pm.Requests)*100)
		} else {
			http["error_rate_5xx"] = "0.00%"
		}
		status["http_metrics"] = http
	}

	return status, nil
}
