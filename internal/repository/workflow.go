package repository

import (
	"database/sql"
	"fmt"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"

	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type WorkflowRepository struct {
	db    *sql.DB
	clock core.Clock
}

// WorkflowOverviewRow holds grouped counts by executor_group and workflow_type
type WorkflowOverviewRow struct {
	ExecutorGroup   string
	WorkflowType    string
	NewCount        int
	ScheduledCount  int
	ExecutingCount  int
	FinishedCount   int
	InProgressCount int
}

// DefinitionStateRow holds counts by state for a workflow type
type DefinitionStateRow struct {
	State           string
	NewCount        int
	ScheduledCount  int
	ExecutingCount  int
	InProgressCount int
	FinishedCount   int
}

const ALL_COLUMNS = ` id, status, execution_count, retry_count, created, modified,
		       next_activation, started, executor_id, executor_group,
		       workflow_type, external_id, business_key, state, state_vars `

func NewWorkflowRepository(db *sql.DB, clock core.Clock) *WorkflowRepository {
	return &WorkflowRepository{db: db, clock: clock}
}

func (r *WorkflowRepository) FindByID(id int64) (*domain.Workflow, error) {
	query := `
		SELECT ` + ALL_COLUMNS + `
		FROM workflow WHERE id = ` + placeholder(1) + `
	`

	var wf domain.Workflow
	err := r.db.QueryRow(query, id).Scan(
		&wf.ID,
		&wf.Status,
		&wf.ExecutionCount,
		&wf.RetryCount,
		&wf.Created,
		&wf.Modified,
		&wf.NextActivation,
		&wf.Started,
		&wf.ExecutorID,
		&wf.ExecutorGroup,
		&wf.WorkflowType,
		&wf.ExternalID,
		&wf.BusinessKey,
		&wf.State,
		&wf.StateVars,
	)

	if err != nil {
		return nil, err
	}
	// If SQLite, convert all timestamps to local
	if config.GetSystemSettingString(config.DATABASE_TYPE) == config.DATABASE_TYPE_SQLLITE {
		wf.Created = wf.Created
		wf.Modified = wf.Modified
		wf.NextActivation = (wf.NextActivation)
		wf.Started = (wf.Started)
	}
	return &wf, nil
}

// helper to force time.Time to local
func toLocalSqlTime(t sql.NullTime) sql.NullTime {
	if !t.Valid {
		return sql.NullTime{}
	}
	if t.Time.IsZero() {
		return t
	}
	t.Time = t.Time.Local()
	return t
}

func (r *WorkflowRepository) Save(wf *domain.Workflow) (int64, error) {
	// Build dialect-aware placeholders
	vals := []interface{}{wf.Status, wf.ExecutionCount, wf.RetryCount, formatDateInDatabase(wf.Created), formatDateInDatabase(wf.Modified), formatDateInDatabaseNull(wf.NextActivation), formatDateInDatabaseNull(wf.Started), wf.ExecutorID, wf.ExecutorGroup, wf.WorkflowType, wf.ExternalID, wf.BusinessKey, wf.State,
		wf.StateVars}
	pps := make([]string, 0, len(vals))
	for i := range vals {
		pps = append(pps, placeholder(i+1))
	}
	base := `INSERT INTO workflow (
		status, execution_count, retry_count, created, modified,
		next_activation, started, executor_id, executor_group,
		workflow_type, external_id, business_key, state, state_vars
	) VALUES (` + strings.Join(pps, ", ") + `)`
	var err error
	if supportsReturning() {
		query := base + " RETURNING id"
		err = r.db.QueryRow(query, vals...).Scan(&wf.ID)
	} else {
		res, e := r.db.Exec(base, vals...)
		if e != nil {
			err = e
		} else {
			id, e2 := res.LastInsertId()
			if e2 != nil {
				err = e2
			} else {
				wf.ID = id
			}
		}
	}
	return wf.ID, err
}

func formatDateInDatabase(created time.Time) string {
	if config.GetSystemSettingString(config.DATABASE_TYPE) == config.DATABASE_TYPE_SQLLITE {
		return created.UTC().Format("2006-01-02 15:04:05.000")
	}
	if config.GetSystemSettingString(config.DATABASE_TYPE) == config.DATABASE_TYPE_MYSQL {
		return created.UTC().Format("2006-01-02 15:04:05.000000")
	}
	// PostgreSQL supports RFC3339
	return created.UTC().Format(time.RFC3339Nano)

}
func formatDateInDatabaseNull(created sql.NullTime) interface{} {
	if !created.Valid {
		return nil
	}

	if config.GetSystemSettingString(config.DATABASE_TYPE) == config.DATABASE_TYPE_SQLLITE {
		// Format as string for SQLite
		return created.Time.UTC().Format("2006-01-02 15:04:05.000")
	}

	// MySQL also needs string format (without T and Z)
	if config.GetSystemSettingString(config.DATABASE_TYPE) == config.DATABASE_TYPE_MYSQL {
		return created.Time.UTC().Format("2006-01-02 15:04:05.000000")
	}

	// Return time.Time directly for PostgreSQL
	return created.Time

}

func (r *WorkflowRepository) FindPendingWorkflows(size int, executorGroup string) (*[]domain.Workflow, error) {
	query := `
		SELECT ` + ALL_COLUMNS + `
		FROM workflow
		WHERE  ` + dateBeforeNow("next_activation") + `
		  AND status in ('NEW', 'IN_PROGRESS')
		  AND executor_id IS NULL
		  AND executor_group = ` + placeholder(1) + `
		ORDER BY next_activation ASC
		LIMIT ` + placeholder(2) + `
	`

	rows, err := r.db.Query(query, executorGroup, size)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []domain.Workflow
	for rows.Next() {
		var wf domain.Workflow
		err := rows.Scan(
			&wf.ID,
			&wf.Status,
			&wf.ExecutionCount,
			&wf.RetryCount,
			&wf.Created,
			&wf.Modified,
			&wf.NextActivation,
			&wf.Started,
			&wf.ExecutorID,
			&wf.ExecutorGroup,
			&wf.WorkflowType,
			&wf.ExternalID,
			&wf.BusinessKey,
			&wf.State,
			&wf.StateVars,
		)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, wf)
	}

	return &workflows, nil
}

func (r *WorkflowRepository) MarkWorkflowAsScheduledForExecution(id int64, executorId int64, modified time.Time) bool {

	query := `
		UPDATE workflow
		SET status = 'SCHEDULED', modified = ` + nowFunc() + `, executor_id = ` + placeholder(1) + `
		WHERE id = ` + placeholder(2) + ` AND modified = ` + placeholder(3) + ` AND status IN ('NEW', 'IN_PROGRESS') AND executor_id IS NULL
	`
	stringdate := formatDateInDatabase(modified)
	result, err := r.db.Exec(query, executorId, id, stringdate)
	if err != nil {
		slog.Error("Failed to mark workflow as scheduled", "error", err, "id", id, "executorId", executorId, "modified", modified)
		return false
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false
	}
	return rowsAffected == 1
}

func (r *WorkflowRepository) UpdateState(id int64, state string) error {

	query := `
		UPDATE workflow
		SET state = ` + placeholder(1) + `, modified = ` + nowFunc() + `, retry_count = 0
		WHERE id = ` + placeholder(2) + `
	`
	_, err := r.db.Exec(query, state, id)
	return err
}

func (r *WorkflowRepository) UpdateWorkflowStatus(id int64, status string) error {
	query := `
		UPDATE workflow
		SET status = ` + placeholder(1) + `, modified = ` + nowFunc() + `
		WHERE id = ` + placeholder(2) + `
	`
	_, err := r.db.Exec(query, status, id)
	return err
}
func (r *WorkflowRepository) UpdateWorkflowStartingTime(id int64) error {
	query := `
		UPDATE workflow
		SET  started = ` + nowFunc() + `
		WHERE id = ` + placeholder(1) + `
	`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *WorkflowRepository) SaveWorkflowVariables(id int64, vars string) error {
	query := `
		UPDATE workflow
		SET state_vars = ` + placeholder(1) + `
		WHERE id = ` + placeholder(2) + `
	`
	_, err := r.db.Exec(query, vars, id)
	return err
}

// SaveWorkflowVariablesAndTouch updates state_vars and touches modified timestamp.
func (r *WorkflowRepository) SaveWorkflowVariablesAndTouch(id int64, vars string) error {
	query := `
		UPDATE workflow
		SET state_vars = ` + placeholder(1) + `, modified = ` + nowFunc() + `
		WHERE id = ` + placeholder(2) + `
	`
	_, err := r.db.Exec(query, vars, id)
	return err
}

func (r *WorkflowRepository) UpdateNextActivationSpecific(id int64, next time.Time) error {
	query := `
		UPDATE workflow
		SET status = 'IN_PROGRESS', next_activation = ` + placeholder(1) + `, modified = ` + nowFunc() + `
		WHERE id = ` + placeholder(2) + `
	`
	_, err := r.db.Exec(query, formatDateInDatabase(next), id)
	return err
}
func (r *WorkflowRepository) UpdateNextActivationOffset(id int64, offset string) error {
	//if supportsReturning() {
	//	query := `
	//		UPDATE workflow
	//		SET status = 'IN_PROGRESS', next_activation = NOW() + ` + placeholder(1) + `::interval,
	//		    modified = ` + nowFunc() + `
	//		WHERE id = ` + placeholder(2) + `
	//	`
	//	_, err := r.db.Exec(query, offset, id)
	//	return err
	//}
	// Non-Postgres: compute next_activation in Go
	var dur time.Duration
	var err error
	dur, err = ParsePostgresInterval(offset)
	if err != nil {
		// try to parse as integer minutes from string like "5" or "5 minutes"
		mins := 0
		fmt.Sscanf(offset, "%d", &mins)
		dur = time.Duration(mins) * time.Minute
	}
	next := time.Now().UTC().Add(dur)
	query := `
		UPDATE workflow
		SET status = 'IN_PROGRESS', next_activation = ` + placeholder(1) + `, modified = ` + nowFunc() + `
		WHERE id = ` + placeholder(2) + `
	`
	_, err = r.db.Exec(query, formatDateInDatabase(next), id)
	return err
}

// ParsePostgresInterval converts a PostgreSQL interval string to time.Duration
func ParsePostgresInterval(interval string) (time.Duration, error) {
	interval = strings.TrimSpace(interval)
	if interval == "" {
		return 0, nil
	}

	// Regex to capture number + unit (hours, minutes, seconds, milliseconds)
	re := regexp.MustCompile(`(?i)(-?\d+(?:\.\d*)?)\s*(hour|hours|minute|minutes|second|seconds|ms|millisecond|milliseconds)`)
	matches := re.FindAllStringSubmatch(interval, -1)
	if matches == nil {
		return 0, fmt.Errorf("invalid interval format: %s", interval)
	}

	var total time.Duration

	for _, m := range matches {
		valueStr, unit := m[1], strings.ToLower(m[2])
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number in interval: %s", valueStr)
		}

		switch unit {
		case "hour", "hours":
			total += time.Duration(value * float64(time.Hour))
		case "minute", "minutes":
			total += time.Duration(value * float64(time.Minute))
		case "second", "seconds":
			total += time.Duration(value * float64(time.Second))
		case "ms", "millisecond", "milliseconds":
			total += time.Duration(value * float64(time.Millisecond))
		default:
			return 0, fmt.Errorf("unknown unit in interval: %s", unit)
		}
	}

	return total, nil
}

func (r *WorkflowRepository) ClearExecutorId(id int64) error {
	query := `
		UPDATE workflow
		SET executor_id = NULL, modified = ` + nowFunc() + `
		WHERE id = ` + placeholder(1) + `
	`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *WorkflowRepository) IncrementRetryCounterAndSetNextActivation(id int64, activation time.Time) error {
	query := `
		UPDATE workflow
		SET status = 'IN_PROGRESS', executor_id = NULL, retry_count = retry_count + 1, next_activation = ` + placeholder(1) + `, modified = ` + nowFunc() + `
		WHERE id = ` + placeholder(2) + `
	`
	_, err := r.db.Exec(query, formatDateInDatabase(activation), id)
	return err
}

func (r *WorkflowRepository) FindByExternalId(id string) (*domain.Workflow, error) {
	query := `
		SELECT ` + ALL_COLUMNS + `
		FROM workflow WHERE external_id = ` + placeholder(1) + `
	`
	var wf domain.Workflow
	err := r.db.QueryRow(query, id).Scan(
		&wf.ID,
		&wf.Status,
		&wf.ExecutionCount,
		&wf.RetryCount,
		&wf.Created,
		&wf.Modified,
		&wf.NextActivation,
		&wf.Started,
		&wf.ExecutorID,
		&wf.ExecutorGroup,
		&wf.WorkflowType,
		&wf.ExternalID,
		&wf.BusinessKey,
		&wf.State,
		&wf.StateVars,
	)
	if err != nil {
		return nil, err
	}
	return &wf, nil
}

func (r *WorkflowRepository) FindStuckWorkflows(minutesRepair string, executorGroup string, limit int) (*[]domain.Workflow, error) {
	var query string
	//if supportsReturning() { // Postgres flavor using interval
	//	query = `
	//	SELECT ` + ALL_COLUMNS + `
	//	FROM workflow
	//	WHERE modified < NOW() - (` + placeholder(1) + ` || ' minutes')::interval
	//	  AND status IN ('SCHEDULED', 'EXECUTING', 'IN_PROGRESS', 'LOCK')
	//	  AND executor_group = ` + placeholder(2) + `
	//	  AND executor_id NOT IN (
	//	      SELECT id
	//	      FROM executors
	//	      WHERE last_active > NOW() - (` + placeholder(1) + ` || ' minutes')::interval
	//	  )
	//	ORDER BY next_activation ASC
	//	LIMIT ` + placeholder(3) + `
	//	`
	//} else {
	// Generic flavor without interval math: compare against parameterized cutoff times
	query = `
		SELECT ` + ALL_COLUMNS + `
		FROM workflow
		WHERE modified < ` + placeholder(1) + `
		  AND status IN ('SCHEDULED', 'EXECUTING', 'IN_PROGRESS', 'LOCK')
		  AND executor_group = ` + placeholder(2) + `
		  AND executor_id NOT IN (
		      SELECT id
		      FROM executors
		      WHERE last_active > ` + placeholder(3) + `
		  )
		ORDER BY next_activation ASC
		LIMIT ` + placeholder(4) + `
		`
	// For non-Postgres, we will shift args: compute cutoffs in Go: now - minutesRepair minutes
	//}
	//if supportsReturning() {
	//	rows, err := r.db.Query(query, minutesRepair, executorGroup, limit)
	//	if err != nil {
	//		return nil, err
	//	}
	//	defer rows.Close()
	//	var workflows []domain.Workflow
	//	for rows.Next() {
	//		var wf domain.Workflow
	//		err := rows.Scan(
	//			&wf.ID,
	//			&wf.Status,
	//			&wf.ExecutionCount,
	//			&wf.RetryCount,
	//			&wf.Created,
	//			&wf.Modified,
	//			&wf.NextActivation,
	//			&wf.Started,
	//			&wf.ExecutorID,
	//			&wf.ExecutorGroup,
	//			&wf.WorkflowType,
	//			&wf.ExternalID,
	//			&wf.BusinessKey,
	//			&wf.State,
	//			&wf.StateVars,
	//		)
	//		if err != nil {
	//			return nil, err
	//		}
	//		workflows = append(workflows, wf)
	//	}
	//	return &workflows, nil
	//} else {
	// minutesRepair is a string like "5" or "5 minutes"; extract leading integer minutes
	mins := 0
	fmt.Sscanf(minutesRepair, "%d", &mins)
	cutoff := time.Now().UTC().Add(-time.Duration(mins) * time.Minute)
	lastActiveCutoff := cutoff
	rows, err := r.db.Query(query, cutoff, executorGroup, lastActiveCutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workflows []domain.Workflow
	for rows.Next() {
		var wf domain.Workflow
		err := rows.Scan(
			&wf.ID,
			&wf.Status,
			&wf.ExecutionCount,
			&wf.RetryCount,
			&wf.Created,
			&wf.Modified,
			&wf.NextActivation,
			&wf.Started,
			&wf.ExecutorID,
			&wf.ExecutorGroup,
			&wf.WorkflowType,
			&wf.ExternalID,
			&wf.BusinessKey,
			&wf.State,
			&wf.StateVars,
		)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, wf)
	}
	return &workflows, nil
	//}
}

func (r *WorkflowRepository) LockWorkflowByModified(id int64, modified time.Time) bool {
	query := `
		UPDATE workflow
		SET status = 'LOCK', executor_id = NULL, retry_count = retry_count + 1, next_activation = ` + placeholder(1) + `, modified = ` + nowFunc() + `
		WHERE id = ` + placeholder(2) + ` AND modified = ` + placeholder(3) + `
	`
	result, err := r.db.Exec(query, formatDateInDatabase(modified), id, formatDateInDatabase(modified))
	if err != nil {
		return false
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false
	}
	return rowsAffected == 1
}
func (r *WorkflowRepository) SearchWorkflows(req models.SearchWorkflowRequest) (*[]domain.Workflow, error) {

	whereClause, args := buildWhereClause(req)

	query := `
		SELECT ` + ALL_COLUMNS + `
		FROM workflow
		` + whereClause +
		` ORDER BY id DESC
	` + buildLimitsAndOffset(req)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []domain.Workflow
	for rows.Next() {
		var wf domain.Workflow
		err := rows.Scan(
			&wf.ID,
			&wf.Status,
			&wf.ExecutionCount,
			&wf.RetryCount,
			&wf.Created,
			&wf.Modified,
			&wf.NextActivation,
			&wf.Started,
			&wf.ExecutorID,
			&wf.ExecutorGroup,
			&wf.WorkflowType,
			&wf.ExternalID,
			&wf.BusinessKey,
			&wf.State,
			&wf.StateVars,
		)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, wf)
	}

	return &workflows, nil
}

// GetWorkflowOverview returns aggregated counts grouped by executor_group and workflow_type
func (r *WorkflowRepository) GetWorkflowOverview() ([]WorkflowOverviewRow, error) {
	query := `
SELECT
    executor_group,
    workflow_type,
    SUM(CASE WHEN status = 'NEW' THEN 1 ELSE 0 END) AS new_count,
    SUM(CASE WHEN status = 'SCHEDULED'  THEN 1 ELSE 0 END) AS scheduled_count,
    SUM(CASE WHEN status = 'EXECUTING' THEN 1 ELSE 0 END) AS executing_count,
    SUM(CASE WHEN status = 'FINISHED'  THEN 1 ELSE 0 END) AS finished_count,
    SUM(CASE WHEN status = 'IN_PROGRESS'  THEN 1 ELSE 0 END) AS in_progress_count
FROM workflow
GROUP BY executor_group, workflow_type;
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []WorkflowOverviewRow
	for rows.Next() {
		var row WorkflowOverviewRow
		if err := rows.Scan(&row.ExecutorGroup, &row.WorkflowType, &row.NewCount, &row.ScheduledCount, &row.ExecutingCount, &row.FinishedCount, &row.InProgressCount); err != nil {
			return nil, err
		}
		res = append(res, row)
	}
	return res, nil
}

// GetDefinitionStateOverview returns counts by state for a given workflow type
func (r *WorkflowRepository) GetDefinitionStateOverview(workflowType string) ([]DefinitionStateRow, error) {
	query := `
SELECT
    COALESCE(state, '') AS state,
    SUM(CASE WHEN status = 'NEW' THEN 1 ELSE 0 END) AS new_count,
    SUM(CASE WHEN status = 'SCHEDULED'  THEN 1 ELSE 0 END) AS scheduled_count,
    SUM(CASE WHEN status = 'EXECUTING' THEN 1 ELSE 0 END) AS executing_count,
    SUM(CASE WHEN status = 'IN_PROGRESS'  THEN 1 ELSE 0 END) AS in_progress_count,
    SUM(CASE WHEN status = 'FINISHED'  THEN 1 ELSE 0 END) AS finished_count
FROM workflow
WHERE workflow_type = ` + placeholder(1) + `
GROUP BY COALESCE(state, '')
ORDER BY COALESCE(state, '')
	`
	rows, err := r.db.Query(query, workflowType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []DefinitionStateRow
	for rows.Next() {
		var row DefinitionStateRow
		if err := rows.Scan(&row.State, &row.NewCount, &row.ScheduledCount, &row.ExecutingCount, &row.InProgressCount, &row.FinishedCount); err != nil {
			return nil, err
		}
		res = append(res, row)
	}
	return res, nil
}

// GetTopExecuting returns workflows currently executing ordered by modified desc
func (r *WorkflowRepository) GetTopExecuting(limit int) (*[]domain.Workflow, error) {
	query := `
		SELECT ` + ALL_COLUMNS + `
		FROM workflow
		WHERE status = 'EXECUTING'
		ORDER BY modified DESC
		LIMIT ` + placeholder(1) + `
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workflows []domain.Workflow
	for rows.Next() {
		var wf domain.Workflow
		if err := rows.Scan(
			&wf.ID,
			&wf.Status,
			&wf.ExecutionCount,
			&wf.RetryCount,
			&wf.Created,
			&wf.Modified,
			&wf.NextActivation,
			&wf.Started,
			&wf.ExecutorID,
			&wf.ExecutorGroup,
			&wf.WorkflowType,
			&wf.ExternalID,
			&wf.BusinessKey,
			&wf.State,
			&wf.StateVars,
		); err != nil {
			return nil, err
		}
		workflows = append(workflows, wf)
	}
	return &workflows, nil
}

// GetNextToExecute returns upcoming workflows with status NEW or IN_PROGRESS ordered by next_activation asc
func (r *WorkflowRepository) GetNextToExecute(limit int) (*[]domain.Workflow, error) {
	query := `
		SELECT ` + ALL_COLUMNS + `
		FROM workflow
		WHERE status IN ('NEW','IN_PROGRESS')
		ORDER BY next_activation ASC
		LIMIT ` + placeholder(1) + `
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workflows []domain.Workflow
	for rows.Next() {
		var wf domain.Workflow
		if err := rows.Scan(
			&wf.ID,
			&wf.Status,
			&wf.ExecutionCount,
			&wf.RetryCount,
			&wf.Created,
			&wf.Modified,
			&wf.NextActivation,
			&wf.Started,
			&wf.ExecutorID,
			&wf.ExecutorGroup,
			&wf.WorkflowType,
			&wf.ExternalID,
			&wf.BusinessKey,
			&wf.State,
			&wf.StateVars,
		); err != nil {
			return nil, err
		}
		workflows = append(workflows, wf)
	}
	return &workflows, nil
}

func buildLimitsAndOffset(req models.SearchWorkflowRequest) string {
	if req.Limit > 0 {
		return fmt.Sprintf(" LIMIT %d OFFSET %d", req.Limit, req.Offset)
	}
	return ""
}
func buildWhereClause(req models.SearchWorkflowRequest) (string, []interface{}) {
	var andClauses []string
	var args []interface{}

	// First, collect the OR-able identity filters: id OR external_id OR business_key
	var orClauses []string
	if req.ID != 0 {
		args = append(args, req.ID)
		orClauses = append(orClauses, fmt.Sprintf("id = %s", placeholder(len(args))))
	}
	if req.ExternalID != "" {
		args = append(args, req.ExternalID)
		orClauses = append(orClauses, fmt.Sprintf("external_id = %s", placeholder(len(args))))
	}
	if req.BusinessKey != "" {
		args = append(args, req.BusinessKey)
		orClauses = append(orClauses, fmt.Sprintf("business_key = %s", placeholder(len(args))))
	}

	// Now, add the remaining AND filters
	if req.ExecutorGroup != "" {
		args = append(args, req.ExecutorGroup)
		andClauses = append(andClauses, fmt.Sprintf("executor_group = %s", placeholder(len(args))))
	}
	if req.WorkflowType != "" {
		args = append(args, req.WorkflowType)
		andClauses = append(andClauses, fmt.Sprintf("workflow_type = %s", placeholder(len(args))))
	}
	if req.State != "" {
		args = append(args, req.State)
		andClauses = append(andClauses, fmt.Sprintf("state = %s", placeholder(len(args))))
	}
	if req.Status != "" {
		args = append(args, req.Status)
		andClauses = append(andClauses, fmt.Sprintf("status = %s", placeholder(len(args))))
	}

	// If there are any OR-able clauses, group them in parentheses and AND with others
	if len(orClauses) > 0 {
		andClauses = append(andClauses, "("+strings.Join(orClauses, " OR ")+")")
	}

	if len(andClauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(andClauses, " AND "), args
}
