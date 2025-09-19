package repository

import (
	"database/sql"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

type WorkflowDefinitionRepository struct {
	db *sql.DB
}

func NewWorkflowDefinitionRepository(db *sql.DB) *WorkflowDefinitionRepository {
	return &WorkflowDefinitionRepository{db: db}
}

// Save inserts a new workflow definition or updates an existing one by name.
// Returns nil on success or an error.
func (r *WorkflowDefinitionRepository) Save(def *domain.WorkflowDefinition) error {
	query := ""
	db := config.GetSystemSettingString(config.DATABASE_TYPE)
	if db == config.DATABASE_TYPE_POSTGRES || db == config.DATABASE_TYPE_SQLLITE {
		query = `
		INSERT INTO workflow_definitions (name, description, created, updated, flow_chart)
		VALUES (` + placeholder(1) + `, ` + placeholder(2) + `, ` + placeholder(3) + `, ` + placeholder(4) + `, ` + placeholder(5) + `)
		ON CONFLICT (name)
		DO UPDATE SET description = EXCLUDED.description,
			updated = EXCLUDED.updated,
			flow_chart = EXCLUDED.flow_chart
	`
	} else if db == config.DATABASE_TYPE_MYSQL {
		query = `
		INSERT INTO workflow_definitions (name, description, created, updated, flow_chart)
		VALUES (` + placeholder(1) + `, ` + placeholder(2) + `, ` + placeholder(3) + `, ` + placeholder(4) + `, ` + placeholder(5) + `)
		ON DUPLICATE KEY UPDATE description = VALUES(description),
			updated = VALUES(updated),
			flow_chart = VALUES(flow_chart)
	`
	} else {
		panic("Unknown database type trying to save workflow definition")
	}

	_, err := r.db.Exec(query, def.Name, def.Description, def.Created, def.Updated, def.FlowChart)
	return err
}

// FindByName fetches a workflow definition by its unique name.
func (r *WorkflowDefinitionRepository) FindByName(name string) (*domain.WorkflowDefinition, error) {
	query := `
		SELECT name, description, created, updated, flow_chart
		FROM workflow_definitions WHERE name = ` + placeholder(1) + `
	`
	var def domain.WorkflowDefinition
	err := r.db.QueryRow(query, name).Scan(
		&def.Name,
		&def.Description,
		&def.Created,
		&def.Updated,
		&def.FlowChart,
	)
	if err != nil {
		return nil, err
	}
	return &def, nil
}

// FindAll returns all workflow definitions.
func (r *WorkflowDefinitionRepository) FindAll() (*[]domain.WorkflowDefinition, error) {
	query := `
		SELECT name, description, created, updated, flow_chart
		FROM workflow_definitions
		ORDER BY name
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	defs := make([]domain.WorkflowDefinition, 0)
	for rows.Next() {
		var d domain.WorkflowDefinition
		if err := rows.Scan(&d.Name, &d.Description, &d.Created, &d.Updated, &d.FlowChart); err != nil {
			return nil, err
		}
		defs = append(defs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &defs, nil
}
