package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type MeetingRepository interface {
	CreateMeeting(ctx context.Context, m domain.Meeting) (*domain.Meeting, error)
	UpdateMeeting(ctx context.Context, m domain.Meeting) error
	CancelMeeting(ctx context.Context, id uuid.UUID) (*domain.Meeting, error)
	DeleteMeeting(ctx context.Context, id uuid.UUID) error
	GetMeetingByID(ctx context.Context, id uuid.UUID) (*domain.Meeting, error)
	ListUserMeetings(ctx context.Context, userID uuid.UUID, from, to sql.NullTime) ([]domain.Meeting, error)
	ListProjectMeetings(ctx context.Context, projectID uuid.UUID) ([]domain.Meeting, error)

	AddParticipant(ctx context.Context, meetingID, userID uuid.UUID, status domain.ParticipantStatus) (*domain.MeetingParticipant, error)
	UpdateParticipantStatus(ctx context.Context, meetingID, userID uuid.UUID, status domain.ParticipantStatus) error
	GetMeetingParticipants(ctx context.Context, meetingID uuid.UUID) ([]domain.MeetingParticipant, error)
	CreateReminder(ctx context.Context, meetingID, userID uuid.UUID, channel domain.ChannelType, reminderTime time.Time) error
}

type meetingRepository struct {
	q *db.Queries
}

func NewMeetingRepository(q *db.Queries) MeetingRepository {
	return &meetingRepository{q: q}
}

func (r *meetingRepository) CreateMeeting(ctx context.Context, m domain.Meeting) (*domain.Meeting, error) {
	pid := ptrToNullUUID(m.ProjectID)
	desc := sql.NullString{}
	if m.Description != nil {
		desc = sql.NullString{String: *m.Description, Valid: true}
	}
	loc := sql.NullString{}
	if m.Location != nil {
		loc = sql.NullString{String: *m.Location, Valid: true}
	}
	row, err := r.q.CreateMeeting(ctx, db.CreateMeetingParams{
		ProjectID:   pid,
		Name:        m.Name,
		Description: desc,
		MeetingType: string(m.Type),
		Location:    loc,
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
		CreatedBy:   m.CreatedBy,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateMeeting", "name", m.Name)
	}
	d := mapDBMeetingToDomain(row)
	return &d, nil
}

func (r *meetingRepository) UpdateMeeting(ctx context.Context, m domain.Meeting) error {
	desc := sql.NullString{}
	if m.Description != nil {
		desc = sql.NullString{String: *m.Description, Valid: true}
	}
	loc := sql.NullString{}
	if m.Location != nil {
		loc = sql.NullString{String: *m.Location, Valid: true}
	}
	pid := ptrToNullUUID(m.ProjectID)
	return errctx.Wrap(r.q.UpdateMeeting(ctx, db.UpdateMeetingParams{
		ID:          m.ID,
		Name:        m.Name,
		Description: desc,
		MeetingType: string(m.Type),
		Location:    loc,
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
		ProjectID:   pid,
	}), "UpdateMeeting", "id", m.ID)
}

func (r *meetingRepository) CancelMeeting(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	row, err := r.q.CancelMeeting(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(err, "CancelMeeting", "id", id)
	}
	d := mapDBMeetingToDomain(row)
	return &d, nil
}

func (r *meetingRepository) DeleteMeeting(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteMeeting(ctx, id), "DeleteMeeting", "id", id)
}

func (r *meetingRepository) GetMeetingByID(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	row, err := r.q.GetMeetingByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetMeetingByID", "id", id)
	}
	d := mapDBMeetingToDomain(row)
	return &d, nil
}

func (r *meetingRepository) ListUserMeetings(ctx context.Context, userID uuid.UUID, from, to sql.NullTime) ([]domain.Meeting, error) {
	params := db.ListUserMeetingsParams{
		UserID:   userID,
		FromTime: from,
		ToTime:   to,
	}
	rows, err := r.q.ListUserMeetings(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "ListUserMeetings", "userID", userID)
	}
	result := make([]domain.Meeting, len(rows))
	for i, m := range rows {
		result[i] = mapDBMeetingToDomain(m)
	}
	return result, nil
}

func (r *meetingRepository) ListProjectMeetings(ctx context.Context, projectID uuid.UUID) ([]domain.Meeting, error) {
	rows, err := r.q.ListProjectMeetings(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjectMeetings", "projectID", projectID)
	}
	result := make([]domain.Meeting, len(rows))
	for i, m := range rows {
		result[i] = mapDBMeetingToDomain(m)
	}
	return result, nil
}

func (r *meetingRepository) AddParticipant(ctx context.Context, meetingID, userID uuid.UUID, status domain.ParticipantStatus) (*domain.MeetingParticipant, error) {
	row, err := r.q.AddMeetingParticipant(ctx, db.AddMeetingParticipantParams{
		MeetingID: meetingID,
		UserID:    userID,
		Status:    string(status),
	})
	if err != nil {
		return nil, errctx.Wrap(err, "AddMeetingParticipant", "meetingID", meetingID, "userID", userID)
	}
	d := mapDBMeetingParticipantToDomain(row)
	return &d, nil
}

func (r *meetingRepository) UpdateParticipantStatus(ctx context.Context, meetingID, userID uuid.UUID, status domain.ParticipantStatus) error {
	return errctx.Wrap(r.q.UpdateParticipantStatus(ctx, db.UpdateParticipantStatusParams{
		MeetingID: meetingID,
		UserID:    userID,
		Status:    string(status),
	}), "UpdateParticipantStatus", "meetingID", meetingID, "userID", userID)
}

func (r *meetingRepository) GetMeetingParticipants(ctx context.Context, meetingID uuid.UUID) ([]domain.MeetingParticipant, error) {
	rows, err := r.q.GetMeetingParticipants(ctx, meetingID)
	if err != nil {
		return nil, errctx.Wrap(err, "GetMeetingParticipants", "meetingID", meetingID)
	}
	result := make([]domain.MeetingParticipant, len(rows))
	for i, p := range rows {
		result[i] = mapDBMeetingParticipantToDomain(p)
	}
	return result, nil
}

func (r *meetingRepository) CreateReminder(_ context.Context, _, _ uuid.UUID, _ domain.ChannelType, _ time.Time) error {
	// meeting_reminders table removed in schema redesign
	return nil
}

func mapDBMeetingToDomain(m db.Meeting) domain.Meeting {
	var desc *string
	if m.Description.Valid {
		desc = &m.Description.String
	}
	var loc *string
	if m.Location.Valid {
		loc = &m.Location.String
	}
	return domain.Meeting{
		ID:          m.ID,
		ProjectID:   nullUUIDToPtr(m.ProjectID),
		Name:        m.Name,
		Description: desc,
		Type:        domain.MeetingType(m.MeetingType),
		Location:    loc,
		Status:      domain.MeetingStatus(m.Status),
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
		CreatedBy:   m.CreatedBy,
	}
}

func mapDBMeetingParticipantToDomain(p db.MeetingParticipant) domain.MeetingParticipant {
	return domain.MeetingParticipant{
		ID:        p.ID,
		MeetingID: p.MeetingID,
		UserID:    p.UserID,
		Status:    domain.ParticipantStatus(p.Status),
	}
}
