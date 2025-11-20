package web

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/internal/controllers"
	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"

	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type WebController struct {
	controllers.AuthController
	manager  *engine.WorkflowManager
	userRepo *repository.UserRepository
}

type searchResultsVM struct {
	Workflows     []workflowRow
	Results       int
	Offset        int64
	Limit         int64
	Q             string
	Status        string
	State         string
	WorkflowType  string
	ExecutorGroup string
	PrevOffset    int64
	NextOffset    int64
}

type searchPageData struct {
	Title         string
	CurrentPath   string
	ResultsData   searchResultsVM
	WorkflowTypes []string
}

type workflowRow struct {
	ID             int64
	ExternalID     string
	WorkflowType   string
	BusinessKey    string
	Status         string
	State          string
	ExecutorGroup  string
	NextActivation string
	Created        string
	Modified       string
}

func NewWebController(manager *engine.WorkflowManager, userRepo *repository.UserRepository) *WebController {
	return &WebController{manager: manager, userRepo: userRepo, AuthController: controllers.AuthController{
		UserRepo: userRepo,
	}}
}

func (wc *WebController) handler(w http.ResponseWriter, r *http.Request) {
	// Define the data to be used in the template
	data := struct {
		Title       string
		Heading     string
		Content     string
		CurrentPath string
	}{
		Title:       "Dashboard",
		Heading:     "Welcome to GopherFlow",
		Content:     "",
		CurrentPath: r.URL.Path,
	}

	// Parse the template file with custom funcs (hasPrefix used in nav)
	tmpl, err := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix}).ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/home.html")
	if err != nil {
		slog.Error("Failed to parse template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the template with the data
	err = tmpl.ExecuteTemplate(w, "home", data)
	if err != nil {
		slog.Error("Failed to execute template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// inProgressTopHandler returns top 10 executing workflows fragment
// overviewHandler returns grouped overview table fragment
func (wc *WebController) overviewHandler(w http.ResponseWriter, r *http.Request) {
	overview, err := wc.manager.Overview()
	if err != nil {
		slog.Error("Failed to load overview", "error", err)
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}
	tmpl, err := template.ParseFS(
		templatesFS,
		"templates/home_overview.html",
	)
	if err != nil {
		slog.Error("Failed to parse overview template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct{ Rows interface{} }{Rows: overview}
	if err := tmpl.ExecuteTemplate(w, "home_overview", data); err != nil {
		slog.Error("Failed to render overview template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (wc *WebController) inProgressTopHandler(w http.ResponseWriter, r *http.Request) {
	items, err := wc.manager.TopExecuting(10)
	if err != nil {
		slog.Error("Failed to load executing workflows", "error", err)
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}
	rows := make([]workflowRow, 0)
	if items != nil {
		for _, wf := range *items {
			rows = append(rows, workflowRow{
				ID:             wf.ID,
				ExternalID:     wf.ExternalID,
				WorkflowType:   wf.WorkflowType,
				BusinessKey:    wf.BusinessKey,
				Status:         wf.Status,
				State:          wf.State,
				ExecutorGroup:  wf.ExecutorGroup,
				NextActivation: getNextActivationString(wf),
				Created:        wf.Created.Local().Format("2006-01-02 15:04:05"),
				Modified:       wf.Modified.Local().Format("2006-01-02 15:04:05"),
			})
		}
	}
	tmpl, err := template.ParseFS(
		templatesFS,
		"templates/home_inprogress.html",
	)
	if err != nil {
		slog.Error("Failed to parse inprogress template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct{ Rows []workflowRow }{Rows: rows}
	if err := tmpl.ExecuteTemplate(w, "home_inprogress", data); err != nil {
		slog.Error("Failed to render inprogress template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// nextExecutionTopHandler returns top 10 upcoming workflows fragment
func (wc *WebController) nextExecutionTopHandler(w http.ResponseWriter, r *http.Request) {
	items, err := wc.manager.NextToExecute(10)
	if err != nil {
		slog.Error("Failed to load next to execute", "error", err)
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}
	rows := make([]workflowRow, 0)
	if items != nil {
		for _, wf := range *items {
			rows = append(rows, workflowRow{
				ID:             wf.ID,
				ExternalID:     wf.ExternalID,
				WorkflowType:   wf.WorkflowType,
				BusinessKey:    wf.BusinessKey,
				Status:         wf.Status,
				State:          wf.State,
				ExecutorGroup:  wf.ExecutorGroup,
				NextActivation: getNextActivationString(wf),
				Created:        wf.Created.Local().Format("2006-01-02 15:04:05"),
				Modified:       wf.Modified.Local().Format("2006-01-02 15:04:05"),
			})
		}
	}
	tmpl, err := template.ParseFS(
		templatesFS,
		"templates/home_next.html",
	)
	if err != nil {
		slog.Error("Failed to parse next template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct{ Rows []workflowRow }{Rows: rows}
	if err := tmpl.ExecuteTemplate(w, "home_next", data); err != nil {
		slog.Error("Failed to render next template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// WorkflowDefinitionModel is a lightweight view model for the definitions list
// to mirror the Java WorkflowDefinitionModel with formatted dates.
type WorkflowDefinitionModel struct {
	Group     string
	Name      string
	FlowChart string
	Created   string
	Updated   string
}

func (wc *WebController) definitionsHandler(w http.ResponseWriter, r *http.Request) {
	defs, err := wc.manager.ListWorkflowDefinitions()
	if err != nil {
		slog.Error("Failed to list workflow definitions", "error", err)
		http.Error(w, "Failed to load definitions", http.StatusInternalServerError)
		return
	}

	models := make([]WorkflowDefinitionModel, 0, len(*defs))
	for _, d := range *defs {
		models = append(models, WorkflowDefinitionModel{
			Name:      d.Name,
			FlowChart: d.FlowChart,
			Created:   d.Created.Local().Format("2006-01-02 15:04:05"),
			Updated:   d.Updated.Local().Format("2006-01-02 15:04:05"),
		})
	}

	data := struct {
		Title       string
		CurrentPath string
		RequestURI  string
		Definitions []WorkflowDefinitionModel
	}{
		Title:       "Workflow Definitions",
		CurrentPath: r.URL.Path,
		RequestURI:  r.URL.Path,
		Definitions: models,
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix}).ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/definition/definitions.html",
	)
	if err != nil {
		slog.Error("Failed to parse definitions template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "definitions", data); err != nil {
		slog.Error("Failed to execute definitions template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (wc *WebController) workflowDetailsHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	wf, err := wc.manager.WorkflowRepo.FindByID(int64(id))
	if err != nil {
		slog.Error("Failed to get workflow", "id", id, "error", err)
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Try to get the workflow definition; proceed with empty fields if not found
	def, _ := wc.manager.GetWorkflowDefinitionByName(wf.WorkflowType)

	// Load workflow actions for rendering
	actions, _ := wc.manager.WorkflowActionRepo.FindAllByWorkflowID(int64(id))

	type workflowVM struct {
		ID             int64
		BusinessKey    string
		ExternalId     string
		Status         string
		State          string
		ExecutorID     string
		Created        string
		Modified       string
		NextActivation string
		StartedAt      string
	}
	// Format times safely
	formatTS := func(t time.Time) string { return t.Local().Format("2006-01-02 15:04:05") }
	var nextAct = getNextActivationString(*wf)
	var startedAt string
	if wf.Started.Valid {
		startedAt = formatTS(wf.Started.Time)
	} else {
		startedAt = "-"
	}
	wvm := workflowVM{
		ID:             wf.ID,
		BusinessKey:    wf.BusinessKey,
		ExternalId:     wf.ExternalID,
		Status:         wf.Status,
		State:          wf.State,
		ExecutorID:     wf.ExecutorID.String,
		Created:        formatTS(wf.Created),
		Modified:       formatTS(wf.Modified),
		NextActivation: nextAct,
		StartedAt:      startedAt,
	}

	type defVM struct {
		Name        string
		Description string
		FlowChart   string
		Created     string
		Updated     string
	}
	var dvm defVM
	if def != nil {
		dvm = defVM{
			Name:        def.Name,
			Description: def.Description,
			FlowChart:   def.FlowChart,
			Created:     def.Created.Local().Format("2006-01-02 15:04:05"),
			Updated:     def.Updated.Local().Format("2006-01-02 15:04:05"),
		}
	}

	type actionVM struct {
		Type     string
		Name     string
		Text     string
		DateTime string
	}

	actionRows := make([]actionVM, 0)
	if actions != nil {
		for _, a := range *actions {
			actionRows = append(actionRows, actionVM{
				Type:     a.Type,
				Name:     a.Name,
				Text:     a.Text,
				DateTime: a.DateTime.Local().Format("2006-01-02 15:04:05"),
			})
		}
	}

	// Parse state variables JSON (map[string]string)
	stateVars := make(map[string]string)
	if wf.StateVars.Valid && len(wf.StateVars.String) > 0 {
		if err := json.Unmarshal([]byte(wf.StateVars.String), &stateVars); err != nil {
			slog.Warn("Failed to parse state vars", "id", id, "error", err)
		}
	}

	type stateOption struct{ Name string }
	type detailModel struct {
		Title              string
		RequestURI         string
		CurrentPath        string
		Workflow           workflowVM
		WorkflowDefinition defVM
		Actions            []actionVM
		StateVars          map[string]string
		States             []stateOption
	}

	// Build States options from workflow definition if available (fallback: current state only)
	var stateOptions []stateOption
	// Prefer states from the workflow implementation via engine registry
	if inst, err := engine.CreateWorkflowInstance(wc.manager, wf.WorkflowType); err == nil && inst != nil {
		for _, s := range inst.GetAllStates() {
			stateOptions = append(stateOptions, stateOption{Name: s.Name})
		}
	}
	// Fallback: include current state if no states discovered
	if len(stateOptions) == 0 && wf.State != "" {
		stateOptions = []stateOption{{Name: wf.State}}
	}

	data := detailModel{
		Title:              fmt.Sprintf("Workflow %d - %s", wf.ID, wf.WorkflowType),
		RequestURI:         r.URL.Path,
		CurrentPath:        r.URL.Path,
		Workflow:           wvm,
		WorkflowDefinition: dvm,
		Actions:            actionRows,
		StateVars:          stateVars,
		States:             stateOptions,
	}

	// Full page render when not HTMX: include header/nav and wrap content so direct URL has full layout
	tmpl := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix})
	tmpl, err = tmpl.ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/search/details.html",
		"templates/search/details_page.html",
	)
	if err != nil {
		slog.Error("Failed to parse details page templates", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Render to buffer first to avoid partial writes, so we can safely set status on error
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "details_page", data); err != nil {
		slog.Error("Failed to execute details page template", "error", err)
		http.Error(w, "Template render error", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
}

func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func (wc *WebController) definitionByNameHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	def, err := wc.manager.GetWorkflowDefinitionByName(name)
	if err != nil {
		slog.Error("Failed to get workflow definition", "name", name, "error", err)
		http.Error(w, "Definition not found", http.StatusNotFound)
		return
	}
	// Map to a view model with formatted dates
	type defVM struct {
		Name        string
		Description string
		FlowChart   string
		Created     string
		Updated     string
	}
	dvm := defVM{
		Name:        def.Name,
		Description: def.Description,
		FlowChart:   def.FlowChart,
		Created:     def.Created.Local().Format("2006-01-02 15:04:05"),
		Updated:     def.Updated.Local().Format("2006-01-02 15:04:05"),
	}

	// Build Overview rows: list all states from workflow and merge DB counts
	inst, err := engine.CreateWorkflowInstance(wc.manager, def.Name)
	if err != nil {
		slog.Error("Failed to create workflow instance for states", "name", def.Name, "error", err)
	}
	var stateList []string
	if inst != nil {
		for _, st := range inst.GetAllStates() {
			stateList = append(stateList, st.Name)
		}
	}
	// load counts by state from repository
	rows, _ := wc.manager.DefinitionOverview(def.Name)
	counts := make(map[string]struct{ New, Scheduled, Executing, InProgress, Finished int })
	for _, r := range rows {
		counts[r.State] = struct{ New, Scheduled, Executing, InProgress, Finished int }{r.NewCount, r.ScheduledCount, r.ExecutingCount, r.InProgressCount, r.FinishedCount}
	}
	type stateRow struct {
		State      string
		New        int
		Scheduled  int
		Executing  int
		InProgress int
		Finished   int
	}
	stateRows := make([]stateRow, 0)
	var totNew, totSch, totExe, totIn, totFin int
	for _, s := range stateList {
		c := counts[s]
		stateRows = append(stateRows, stateRow{State: s, New: c.New, Scheduled: c.Scheduled, Executing: c.Executing, InProgress: c.InProgress, Finished: c.Finished})
		totNew += c.New
		totSch += c.Scheduled
		totExe += c.Executing
		totIn += c.InProgress
		totFin += c.Finished
	}

	type detailModel struct {
		Title              string
		RequestURI         string
		WorkflowDefinition defVM
		States             []stateRow
		Totals             struct{ New, Scheduled, Executing, InProgress, Finished int }
	}
	var totals = struct{ New, Scheduled, Executing, InProgress, Finished int }{totNew, totSch, totExe, totIn, totFin}
	data := detailModel{
		Title:              "Workflow Definition - " + def.Name,
		RequestURI:         r.URL.Path,
		WorkflowDefinition: dvm,
		States:             stateRows,
		Totals:             totals,
	}

	// For htmx fragment update, only return the inner content fragment
	tmpl, err := template.ParseFS(
		templatesFS,
		"templates/definition/definitionByName.html",
	)
	if err != nil {
		slog.Error("Failed to parse definitionByName template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("Failed to execute definitionByName template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ExecutorModel maps executor info for the view.
type ExecutorModel struct {
	ID        int64
	Group     string
	Host      string
	StartedAt string
	LastAlive string
	CssClass  string
}

func friendlyTimeAgo(since time.Time) string {
	// Interpret the given timestamp as local

	d := time.Since(since)
	if d < 0 {
		d = 0
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func statusCssClass(last time.Time) string {
	mins := time.Since(last).Minutes()
	if mins < 2 {
		return "bg-green-300"
	} else if mins < 10 {
		return "bg-amber-200"
	}
	return "bg-gray-200"
}

func (wc *WebController) searchPageHandler(w http.ResponseWriter, r *http.Request) {
	// Load workflow types for dropdown
	var types []string
	if defs, err := wc.manager.ListWorkflowDefinitions(); err == nil && defs != nil {
		for _, d := range *defs {
			types = append(types, d.Name)
		}
	} else if err != nil {
		slog.Warn("Failed to load workflow definitions for search filter", "error", err)
	}

	data := searchPageData{
		Title:       "Search Workflows",
		CurrentPath: r.URL.Path,
		ResultsData: searchResultsVM{
			Workflows:     nil,
			Results:       0,
			Offset:        0,
			Limit:         50,
			Q:             "",
			Status:        "",
			State:         "",
			WorkflowType:  "",
			ExecutorGroup: "",
			PrevOffset:    0,
			NextOffset:    50,
		},
		WorkflowTypes: types,
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix}).ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/search/search.html",
		"templates/search/results.html",
	)
	if err != nil {
		slog.Error("Failed to parse search template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "search", data); err != nil {
		slog.Error("Failed to execute search template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (wc *WebController) searchResultsHandler(w http.ResponseWriter, r *http.Request) {
	// Build request from query params
	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	state := r.URL.Query().Get("state")
	wfType := r.URL.Query().Get("workflowType")
	execGroup := r.URL.Query().Get("executorGroup")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	var limit int64 = 50
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = int64(v)
		}
	}
	var offset int64 = 0
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			offset = int64(v)
		}
	}
	var id int64
	if q != "" {
		if v, err := strconv.ParseInt(q, 10, 64); err == nil {
			id = v
		}
	}

	req := models.SearchWorkflowRequest{
		ID:            id,
		ExternalID:    q,
		BusinessKey:   q,
		Status:        status,
		State:         state,
		WorkflowType:  wfType,
		ExecutorGroup: execGroup,
		Limit:         limit,
		Offset:        offset,
	}

	results, err := wc.manager.SearchWorkflows(req)
	if err != nil {
		slog.Error("Search failed", "error", err)
		http.Error(w, "Failed to search", http.StatusInternalServerError)
		return
	}
	rows := make([]workflowRow, 0)
	if results != nil {
		for _, wf := range *results {
			// Compute NextActivation string
			nextAct := getNextActivationString(wf)

			rows = append(rows, workflowRow{
				ID:             wf.ID,
				ExternalID:     wf.ExternalID,
				WorkflowType:   wf.WorkflowType,
				BusinessKey:    wf.BusinessKey,
				Status:         wf.Status,
				State:          wf.State,
				ExecutorGroup:  wf.ExecutorGroup,
				NextActivation: nextAct,
				Created:        wf.Created.Local().Format("2006-01-02 15:04:05"),
				Modified:       wf.Modified.Local().Format("2006-01-02 15:04:05"),
			})
		}
	}

	prevOffset := offset - limit
	if prevOffset < 0 {
		prevOffset = 0
	}
	nextOffset := offset + limit

	data := searchResultsVM{
		Workflows:     rows,
		Results:       len(rows),
		Offset:        offset,
		Limit:         limit,
		Q:             q,
		Status:        status,
		State:         state,
		WorkflowType:  wfType,
		ExecutorGroup: execGroup,
		PrevOffset:    prevOffset,
		NextOffset:    nextOffset,
	}

	// Return only the results fragment
	tmpl, err := template.ParseFS(
		templatesFS,
		"templates/search/results.html",
	)
	if err != nil {
		slog.Error("Failed to parse search results template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "search_results", data); err != nil {
		slog.Error("Failed to execute search results template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getNextActivationString(wf domain.Workflow) string {
	var nextAct string
	if wf.Status == "FINISHED" || wf.Status == "FAILED" {
		nextAct = "-"
	} else if wf.NextActivation.Valid {
		t := wf.NextActivation.Time.Local()
		if time.Now().Before(t) {
			// future: show "in X"
			dur := time.Until(t)
			if dur < 0 {
				dur = 0
			}
			if dur < time.Minute {
				nextAct = fmt.Sprintf("in %ds", int(dur.Seconds()))
			} else if dur < time.Hour {
				nextAct = fmt.Sprintf("in %dm", int(dur.Minutes()))
			} else if dur < 24*time.Hour {
				nextAct = fmt.Sprintf("in %dh", int(dur.Hours()))
			} else {
				nextAct = fmt.Sprintf("in %dd", int(dur.Hours()/24))
			}
		} else {
			// past or now: show "X ago"
			nextAct = friendlyTimeAgo(t)
		}
	} else {
		nextAct = "-"
	}
	return nextAct
}

func (wc *WebController) executorsHandler(w http.ResponseWriter, r *http.Request) {
	execs, err := wc.manager.ListExecutors(50)
	if err != nil {
		slog.Error("Failed to list executors", "error", err)
		http.Error(w, "Failed to load executors", http.StatusInternalServerError)
		return
	}
	models := make([]ExecutorModel, 0, len(execs))
	for _, e := range execs {
		models = append(models, ExecutorModel{
			ID:        e.ID,
			Group:     "default", // grouping not modeled yet
			Host:      e.Name,
			StartedAt: e.Started.Local().Format("2006-01-02 15:04:05"),
			LastAlive: friendlyTimeAgo(e.LastActive.Local()),
			CssClass:  statusCssClass(e.LastActive.Local()),
		})
	}
	data := struct {
		Title       string
		CurrentPath string
		Executors   []ExecutorModel
	}{
		Title:       "Executors",
		CurrentPath: r.URL.Path,
		Executors:   models,
	}
	tmpl, err := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix}).ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/executors/executors.html",
	)
	if err != nil {
		slog.Error("Failed to parse executors template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "executors", data); err != nil {
		slog.Error("Failed to execute executors template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// settingsHandler renders the Settings page with key/value pairs from system settings
func (wc *WebController) settingsHandler(w http.ResponseWriter, r *http.Request) {
	// Prepare list of keys from config
	type kv struct{ Key, Value string }
	rows := []kv{
		{Key: "GFLOW_DATABASE_TYPE", Value: config.GetSystemSettingString(config.DATABASE_TYPE)},
		{Key: "GFLOW_ENGINE_SERVER_WEB_PORT", Value: config.GetSystemSettingString(config.ENGINE_SERVER_WEB_PORT)},
		{Key: "GFLOW_ENGINE_CHECK_DB_INTERVAL", Value: config.GetSystemSettingString(config.ENGINE_CHECK_DB_INTERVAL)},
		{Key: "GFLOW_ENGINE_STUCK_WORKFLOWS_INTERVAL", Value: config.GetSystemSettingString(config.ENGINE_STUCK_WORKFLOWS_INTERVAL)},
		{Key: "GFLOW_ENGINE_STUCK_WORKFLOWS_REPAIR_AFTER_MINUTES", Value: config.GetSystemSettingString(config.ENGINE_STUCK_WORKFLOWS_REPAIR_AFTER_MINUTES)},
		{Key: "GFLOW_ENGINE_BATCH_SIZE", Value: config.GetSystemSettingString(config.ENGINE_BATCH_SIZE)},
		{Key: "GFLOW_ENGINE_EXECUTOR_GROUP", Value: config.GetSystemSettingString(config.ENGINE_EXECUTOR_GROUP)},
		{Key: "GFLOW_ENGINE_EXECUTOR_SIZE", Value: config.GetSystemSettingString(config.ENGINE_EXECUTOR_SIZE)},
		{Key: "GFLOW_WEB_SESSION_EXPIRY_HOURS", Value: config.GetSystemSettingString(config.WEB_SESSION_EXPIRY_HOURS)},
	}
	data := struct {
		Title       string
		CurrentPath string
		Rows        []kv
	}{
		Title:       "Settings",
		CurrentPath: r.URL.Path,
		Rows:        rows,
	}
	tmpl, err := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix}).ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/settings.html",
	)
	if err != nil {
		slog.Error("Failed to parse settings template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "settings", data); err != nil {
		slog.Error("Failed to execute settings template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// createWorkflowPageHandler renders the page to create a workflow for a given workflow type (definition name).
func (wc *WebController) createWorkflowPageHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	data := struct {
		Title        string
		CurrentPath  string
		RequestURI   string
		WorkflowType string
	}{
		Title:        "Create Workflow - " + name,
		CurrentPath:  r.URL.Path,
		RequestURI:   r.URL.Path,
		WorkflowType: name,
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix}).ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/definition/create_workflow.html",
	)
	if err != nil {
		slog.Error("Failed to parse create_workflow template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "create_workflow", data); err != nil {
		slog.Error("Failed to execute create_workflow template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// --- Authentication helpers and handlers ---

func (wc *WebController) renderLogin(w http.ResponseWriter, data map[string]any) {
	tmpl, err := template.New("").ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/login.html",
	)
	if err != nil {
		slog.Error("Failed to parse login template", "error", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	if err := tmpl.ExecuteTemplate(w, "login", data); err != nil {
		slog.Error("Failed to render login template", "error", err)
		http.Error(w, "Render error", http.StatusInternalServerError)
		return
	}
}

func (wc *WebController) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	wc.renderLogin(w, map[string]any{"Title": "Login"})
}

func (wc *WebController) loginSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		wc.renderLogin(w, map[string]any{"Error": "Invalid form"})
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusUnauthorized)
		wc.renderLogin(w, map[string]any{"Error": "Username and password are required"})
		return
	}
	u, err := wc.userRepo.FindByUsername(username)
	if err != nil {
		slog.Error("FindByUsername failed", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		w.WriteHeader(http.StatusUnauthorized)
		wc.renderLogin(w, map[string]any{"Error": "Invalid username or password"})
		return
	}
	// Compare bcrypt hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		wc.renderLogin(w, map[string]any{"Error": "Invalid username or password"})
		return
	}
	// Generate session id
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		slog.Error("rand.Read failed", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	sessionID := hex.EncodeToString(buf)
	expiryHours := config.GetSystemSettingInteger(config.WEB_SESSION_EXPIRY_HOURS)
	expires := time.Now().Add(time.Duration(expiryHours) * time.Hour)
	if err := wc.userRepo.UpdateSession(u.ID, sessionID, expires); err != nil {
		slog.Error("UpdateSession failed", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "sessionId",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

//func (wc *WebController) requireAuth(next http.HandlerFunc) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		if r.URL.Path == "/login" {
//			next(w, r)
//			return
//		}
//		// 1) Try session cookie
//		if c, err := r.Cookie("sessionId"); err == nil && c.Value != "" {
//			u, err := wc.userRepo.FindBySessionID(c.Value, time.Now().UTC())
//			if err == nil && u != nil {
//				next(w, r)
//				return
//			}
//		}
//		// 2) Try API key from headers
//		// Supported headers: X-API-Key: <key> or Authorization: ApiKey <key>
//		apiKey := r.Header.Get("X-API-Key")
//		if apiKey == "" {
//			authz := r.Header.Get("Authorization")
//			if strings.HasPrefix(strings.ToLower(authz), "apikey ") {
//				apiKey = strings.TrimSpace(authz[7:])
//			}
//		}
//		if apiKey != "" {
//			u, err := wc.userRepo.FindByApiKey(apiKey)
//			if err == nil && u != nil {
//				// Proceed as authenticated
//				next(w, r)
//				return
//			}
//		}
//		// Otherwise redirect to login for browser flows
//		http.Redirect(w, r, "/login", http.StatusSeeOther)
//	}
//}

// logoutHandler clears the current user's session and redirects to the login page.
func (wc *WebController) logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get session cookie if exists
	c, err := r.Cookie("sessionId")
	if err == nil && c.Value != "" {
		// Best-effort clear in DB
		if err := wc.userRepo.ClearSessionBySessionID(c.Value); err != nil {
			slog.Warn("Failed to clear session in DB during logout", "error", err)
		}
		// Expire cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "sessionId",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})
	}
	// Always redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// User Management Handlers

// usersHandler displays a list of all users
func (wc *WebController) usersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := wc.userRepo.FindAll()
	if err != nil {
		slog.Error("Failed to get users", "error", err)
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	// Convert to view model with masked passwords
	type userVM struct {
		ID         int64
		Username   string 
		ApiKey     string
		Created    string
		Enabled    string
		SessionID  string
	}

	userList := make([]userVM, 0)
	if users != nil {
		for _, u := range *users {
			var apiKey, created, enabled, sessionID string
			
			if u.ApiKey.Valid {
				apiKey = u.ApiKey.String
			}
			
			if u.Created.Valid {
				created = u.Created.Time.Local().Format("2006-01-02 15:04:05")
			}
			
			if u.Enabled.Valid {
				if u.Enabled.Bool {
					enabled = "Yes"
				} else {
					enabled = "No"
				}
			}
			
			if u.SessionID.Valid {
				sessionID = "Active"
			} else {
				sessionID = "None"
			}
			
			userList = append(userList, userVM{
				ID:         u.ID,
				Username:   u.Username,
				ApiKey:     apiKey,
				Created:    created,
				Enabled:    enabled,
				SessionID:  sessionID,
			})
		}
	}

	data := struct {
		Title       string
		CurrentPath string
		Users       []userVM
	}{
		Title:       "User Management",
		CurrentPath: r.URL.Path,
		Users:       userList,
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix}).ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/users/users.html",
	)
	if err != nil {
		slog.Error("Failed to parse users template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "users", data); err != nil {
		slog.Error("Failed to execute users template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// createUserHandler displays the form to create a new user
func (wc *WebController) createUserHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title       string
		CurrentPath string
	}{
		Title:       "Create User",
		CurrentPath: r.URL.Path,
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{"hasPrefix": hasPrefix}).ParseFS(
		templatesFS,
		"templates/fragments/header.html",
		"templates/fragments/nav.html",
		"templates/users/create_user.html",
	)
	if err != nil {
		slog.Error("Failed to parse create user template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "create_user", data); err != nil {
		slog.Error("Failed to execute create user template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// createUserSubmitHandler processes the form submission to create a new user
func (wc *WebController) createUserSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}
	
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	apiKey := r.FormValue("apiKey")
	enabledStr := r.FormValue("enabled")
	
	if username == "" || password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	
	// Check if username already exists
	existingUser, err := wc.userRepo.FindByUsername(username)
	if err != nil {
		slog.Error("Error checking for existing user", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if existingUser != nil {
		http.Error(w, "Username already exists", http.StatusBadRequest)
		return
	}
	
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	
	// Create the user
	user := &domain.User{
		Username: username,
		Password: string(hashedPassword),
		Created:  sql.NullTime{Time: time.Now().UTC(), Valid: true},
		Enabled:  sql.NullBool{Bool: enabledStr == "on", Valid: true},
	}
	
	if apiKey != "" {
		user.ApiKey = sql.NullString{String: apiKey, Valid: true}
	}
	
	_, err = wc.userRepo.Save(user)
	if err != nil {
		slog.Error("Failed to create user", "error", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	
	// Redirect to users list
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

// deleteUserHandler processes a request to delete a user
func (wc *WebController) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	
	// Don't allow deleting the currently logged-in user
	c, err := r.Cookie("sessionId")
	if err == nil && c.Value != "" {
		currentUser, _ := wc.userRepo.FindBySessionID(c.Value, time.Now().UTC())
		if currentUser != nil && currentUser.ID == id {
			http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
			return
		}
	}
	
	// Check if user exists
	user, err := wc.userRepo.FindById(id)
	if err != nil {
		slog.Error("Error finding user", "id", id, "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	// Delete the user
	err = wc.userRepo.DeleteById(id)
	if err != nil {
		slog.Error("Failed to delete user", "id", id, "error", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	
	// Redirect to users list
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}
