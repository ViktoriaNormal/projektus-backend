package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

// System field categories.
// Text-like system fields: value stored directly as text on the tasks row.
var systemTextColumn = map[uuid.UUID]string{
	domain.SystemBoardFieldIDs["priority"]:   "t.priority",
	domain.SystemBoardFieldIDs["estimation"]: "t.estimation",
	domain.SystemBoardFieldIDs["status"]:     "t.column_id::text",
}

// User-type system fields: tasks column stores a member_id (FK to members),
// but filter values are user_ids — need to resolve via members table.
var systemUserColumn = map[uuid.UUID]string{
	domain.SystemBoardFieldIDs["author"]:   "t.owner_id",
	domain.SystemBoardFieldIDs["assignee"]: "t.executor_id",
}

// Watchers system field — stored in task_watchers table with member_ids.
var systemWatchersID = domain.SystemBoardFieldIDs["watchers"]

// isSystemField returns true for any recognized system field UUID.
func isSystemField(id uuid.UUID) bool {
	if _, ok := systemTextColumn[id]; ok {
		return true
	}
	if _, ok := systemUserColumn[id]; ok {
		return true
	}
	return id == systemWatchersID
}

// BuildTaskFilter returns the set of task IDs on the given board matching all field filters.
// AND semantics between different fields, OR within values of one field.
// Returns nil when filters is empty (meaning "no filter applied").
func BuildTaskFilter(ctx context.Context, dbtx db.DBTX, projectID, boardID uuid.UUID, filters map[string][]string) (map[uuid.UUID]struct{}, error) {
	if len(filters) == 0 {
		return nil, nil
	}

	tagValues := filters["__tags__"]

	// Collect field-value filters (excluding __tags__), split into system vs custom.
	type fieldFilter struct {
		id     uuid.UUID
		values []string
	}
	var systemFilters []fieldFilter
	var customFilters []fieldFilter
	for k, v := range filters {
		if k == "__tags__" || len(v) == 0 {
			continue
		}
		fid, err := uuid.Parse(k)
		if err != nil {
			continue
		}
		if isSystemField(fid) {
			systemFilters = append(systemFilters, fieldFilter{id: fid, values: v})
		} else {
			customFilters = append(customFilters, fieldFilter{id: fid, values: v})
		}
	}

	// Fetch field types only for custom field filters (system fields are not in board_fields).
	fieldTypes := make(map[uuid.UUID]string)
	if len(customFilters) > 0 {
		ids := make([]uuid.UUID, len(customFilters))
		for i, f := range customFilters {
			ids[i] = f.id
		}
		var err error
		fieldTypes, err = queryFieldTypes(ctx, dbtx, boardID, ids)
		if err != nil {
			return nil, fmt.Errorf("query field types: %w", err)
		}
	}

	// Build dynamic SQL
	args := []interface{}{projectID, boardID}
	argIdx := 3
	q := `SELECT DISTINCT t.id FROM tasks t WHERE t.project_id = $1 AND t.board_id = $2 AND t.deleted_at IS NULL`

	// System field filters.
	for _, ff := range systemFilters {
		placeholders := make([]string, len(ff.values))
		for i, v := range ff.values {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		inList := strings.Join(placeholders, ",")

		if col, ok := systemTextColumn[ff.id]; ok {
			// Text system fields (priority, estimation, status) — direct column match.
			q += fmt.Sprintf(` AND %s IN (%s)`, col, inList)
		} else if col, ok := systemUserColumn[ff.id]; ok {
			// User system fields (author, assignee) — column stores member_id,
			// filter values are user_ids → resolve via members table.
			q += fmt.Sprintf(
				` AND EXISTS (SELECT 1 FROM members m WHERE m.id = %s AND m.user_id::text IN (%s))`,
				col, inList,
			)
		} else if ff.id == systemWatchersID {
			// Watchers — stored in task_watchers with member_ids.
			q += fmt.Sprintf(
				` AND EXISTS (SELECT 1 FROM task_watchers tw JOIN members m ON m.id = tw.member_id WHERE tw.task_id = t.id AND m.user_id::text IN (%s))`,
				inList,
			)
		}
	}

	// Custom field filters — via task_field_values.
	for _, ff := range customFilters {
		ft := fieldTypes[ff.id]
		args = append(args, ff.id)
		fieldArgIdx := argIdx
		argIdx++

		placeholders := make([]string, len(ff.values))
		for i, v := range ff.values {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		inList := strings.Join(placeholders, ",")

		switch ft {
		case "user":
			// Custom user field: value_text stores member_id, filter values are user_ids.
			q += fmt.Sprintf(
				` AND EXISTS (SELECT 1 FROM task_field_values tfv JOIN members m ON m.id::text = tfv.value_text WHERE tfv.task_id = t.id AND tfv.field_id = $%d AND m.user_id::text IN (%s))`,
				fieldArgIdx, inList,
			)
		case "user_list":
			// Custom user_list field: value_json is a JSON array of member_ids.
			q += fmt.Sprintf(
				` AND EXISTS (SELECT 1 FROM task_field_values tfv, jsonb_array_elements_text(COALESCE(tfv.value_json, '[]'::jsonb)) AS elem WHERE tfv.task_id = t.id AND tfv.field_id = $%d AND EXISTS (SELECT 1 FROM members m WHERE m.id::text = elem AND m.user_id::text IN (%s)))`,
				fieldArgIdx, inList,
			)
		case "multiselect":
			q += fmt.Sprintf(
				` AND EXISTS (SELECT 1 FROM task_field_values tfv, jsonb_array_elements_text(COALESCE(tfv.value_json, '[]'::jsonb)) AS elem WHERE tfv.task_id = t.id AND tfv.field_id = $%d AND elem IN (%s))`,
				fieldArgIdx, inList,
			)
		default:
			q += fmt.Sprintf(
				` AND EXISTS (SELECT 1 FROM task_field_values tfv WHERE tfv.task_id = t.id AND tfv.field_id = $%d AND tfv.value_text IN (%s))`,
				fieldArgIdx, inList,
			)
		}
	}

	// Tags filter.
	if len(tagValues) > 0 {
		placeholders := make([]string, len(tagValues))
		for i, v := range tagValues {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		q += fmt.Sprintf(
			` AND EXISTS (SELECT 1 FROM task_tags tt JOIN tags tag ON tag.id = tt.tag_id WHERE tt.task_id = t.id AND tag.name IN (%s))`,
			strings.Join(placeholders, ","),
		)
	}

	rows, err := dbtx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("filter tasks: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = struct{}{}
	}
	return result, rows.Err()
}

// BuildBoardFilter returns the set of all task IDs on the given board (no field-level filtering).
func BuildBoardFilter(ctx context.Context, dbtx db.DBTX, projectID, boardID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	rows, err := dbtx.QueryContext(ctx,
		`SELECT id FROM tasks WHERE project_id = $1 AND board_id = $2 AND deleted_at IS NULL`,
		projectID, boardID,
	)
	if err != nil {
		return nil, fmt.Errorf("board filter: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = struct{}{}
	}
	return result, rows.Err()
}

func queryFieldTypes(ctx context.Context, dbtx db.DBTX, boardID uuid.UUID, fieldIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	placeholders := make([]string, len(fieldIDs))
	args := make([]interface{}, 0, len(fieldIDs)+1)
	args = append(args, boardID)
	for i, id := range fieldIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	q := fmt.Sprintf(`SELECT id, field_type FROM board_fields WHERE board_id = $1 AND id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := dbtx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]string)
	for rows.Next() {
		var id uuid.UUID
		var ft string
		if err := rows.Scan(&id, &ft); err != nil {
			return nil, err
		}
		result[id] = ft
	}
	return result, rows.Err()
}

// buildScrumFilter resolves board + field filters for Scrum analytics.
// When boardID is set (even without field filters), returns all task IDs on that board.
// When only field filters are set, resolves the default board first.
func (s *ScrumAnalyticsService) buildScrumFilter(ctx context.Context, projectID uuid.UUID, boardID *uuid.UUID, fieldFilters map[string][]string) (map[uuid.UUID]struct{}, error) {
	if boardID == nil && len(fieldFilters) == 0 {
		return nil, nil
	}

	// Resolve effective board ID
	var bid uuid.UUID
	if boardID != nil {
		bid = *boardID
	} else {
		board, err := s.queries.GetDefaultBoardForProject(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("не удалось найти доску проекта: %w", err)
		}
		bid = board.ID
	}

	if len(fieldFilters) > 0 {
		return BuildTaskFilter(ctx, s.dbtx, projectID, bid, fieldFilters)
	}
	// Only board_id filtering, no field filters
	return BuildBoardFilter(ctx, s.dbtx, projectID, bid)
}

// --- In-memory filter helpers ---

func filterCompletedTasks(tasks []completedTask, set map[uuid.UUID]struct{}) []completedTask {
	result := make([]completedTask, 0, len(tasks))
	for _, t := range tasks {
		if _, ok := set[t.TaskID]; ok {
			result = append(result, t)
		}
	}
	return result
}

func filterHistoryRows(rows []db.GetProjectTaskHistoryForKanbanRow, set map[uuid.UUID]struct{}) []db.GetProjectTaskHistoryForKanbanRow {
	result := make([]db.GetProjectTaskHistoryForKanbanRow, 0, len(rows))
	for _, r := range rows {
		if _, ok := set[r.TaskID]; ok {
			result = append(result, r)
		}
	}
	return result
}

func countInSet(ids []uuid.UUID, set map[uuid.UUID]struct{}) int {
	n := 0
	for _, id := range ids {
		if _, ok := set[id]; ok {
			n++
		}
	}
	return n
}

func filterSprintTaskRows(rows []db.GetSprintTasksForAnalyticsRow, set map[uuid.UUID]struct{}) []db.GetSprintTasksForAnalyticsRow {
	result := make([]db.GetSprintTasksForAnalyticsRow, 0, len(rows))
	for _, r := range rows {
		if _, ok := set[r.ID]; ok {
			result = append(result, r)
		}
	}
	return result
}

func filterSprintHistoryRows(rows []db.GetSprintTaskStatusHistoryRow, set map[uuid.UUID]struct{}) []db.GetSprintTaskStatusHistoryRow {
	result := make([]db.GetSprintTaskStatusHistoryRow, 0, len(rows))
	for _, r := range rows {
		if _, ok := set[r.TaskID]; ok {
			result = append(result, r)
		}
	}
	return result
}
