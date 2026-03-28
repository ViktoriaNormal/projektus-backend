package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type MeetingRepository interface {
	CreateMeeting(ctx context.Context, m domain.Meeting) (*domain.Meeting, error)
	UpdateMeeting(ctx context.Context, m domain.Meeting) error
	CancelMeeting(ctx context.Context, id string) (*domain.Meeting, error)
	DeleteMeeting(ctx context.Context, id string) error
	GetMeetingByID(ctx context.Context, id string) (*domain.Meeting, error)
	ListUserMeetings(ctx context.Context, userID string, from, to sql.NullTime) ([]domain.Meeting, error)
	ListProjectMeetings(ctx context.Context, projectID string) ([]domain.Meeting, error)

	AddParticipant(ctx context.Context, meetingID, userID string, status domain.ParticipantStatus) (*domain.MeetingParticipant, error)
	UpdateParticipantStatus(ctx context.Context, meetingID, userID string, status domain.ParticipantStatus) error
	GetMeetingParticipants(ctx context.Context, meetingID string) ([]domain.MeetingParticipant, error)
	CreateReminder(ctx context.Context, meetingID, userID string, channel domain.ChannelType, reminderTime time.Time) error
}

type meetingRepository struct {
	q *db.Queries
}

func NewMeetingRepository(q *db.Queries) MeetingRepository {
	return &meetingRepository{q: q}
}

func (r *meetingRepository) CreateMeeting(ctx context.Context, m domain.Meeting) (*domain.Meeting, error) {
	var pid uuid.NullUUID
	if m.ProjectID != nil {
		if id, err := uuid.Parse(*m.ProjectID); err == nil {
			pid = uuid.NullUUID{UUID: id, Valid: true}
		}
	}
	desc := sql.NullString{}
	if m.Description != nil {
		desc = sql.NullString{String: *m.Description, Valid: true}
	}
	loc := sql.NullString{}
	if m.Location != nil {
		loc = sql.NullString{String: *m.Location, Valid: true}
	}
	creator, err := uuid.Parse(m.CreatedBy)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateMeeting(ctx, db.CreateMeetingParams{
		ProjectID:   pid,
		Name:        m.Name,
		Description: desc,
		MeetingType: string(m.Type),
		Location:    loc,
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
		CreatedBy:   creator,
	})
	if err != nil {
		return nil, err
	}
	d := mapDBMeetingToDomain(row)
	return &d, nil
}

func (r *meetingRepository) UpdateMeeting(ctx context.Context, m domain.Meeting) error {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return err
	}
	desc := sql.NullString{}
	if m.Description != nil {
		desc = sql.NullString{String: *m.Description, Valid: true}
	}
	loc := sql.NullString{}
	if m.Location != nil {
		loc = sql.NullString{String: *m.Location, Valid: true}
	}
	return r.q.UpdateMeeting(ctx, db.UpdateMeetingParams{
		ID:          id,
		Name:        m.Name,
		Description: desc,
		MeetingType: string(m.Type),
		Location:    loc,
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
	})
}

func (r *meetingRepository) CancelMeeting(ctx context.Context, id string) (*domain.Meeting, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CancelMeeting(ctx, uid)
	if err != nil {
		return nil, err
	}
	d := mapDBMeetingToDomain(row)
	return &d, nil
}

func (r *meetingRepository) DeleteMeeting(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteMeeting(ctx, uid)
}

func (r *meetingRepository) GetMeetingByID(ctx context.Context, id string) (*domain.Meeting, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetMeetingByID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	d := mapDBMeetingToDomain(row)
	return &d, nil
}

func (r *meetingRepository) ListUserMeetings(ctx context.Context, userID string, from, to sql.NullTime) ([]domain.Meeting, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	params := db.ListUserMeetingsParams{
		UserID:   uid,
		FromTime: from,
		ToTime:   to,
	}
	rows, err := r.q.ListUserMeetings(ctx, params)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Meeting, len(rows))
	for i, m := range rows {
		result[i] = mapDBMeetingToDomain(m)
	}
	return result, nil
}

func (r *meetingRepository) ListProjectMeetings(ctx context.Context, projectID string) ([]domain.Meeting, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListProjectMeetings(ctx, uuid.NullUUID{UUID: pid, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Meeting, len(rows))
	for i, m := range rows {
		result[i] = mapDBMeetingToDomain(m)
	}
	return result, nil
}

func (r *meetingRepository) AddParticipant(ctx context.Context, meetingID, userID string, status domain.ParticipantStatus) (*domain.MeetingParticipant, error) {
	mid, err := uuid.Parse(meetingID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.AddMeetingParticipant(ctx, db.AddMeetingParticipantParams{
		MeetingID: mid,
		UserID:    uid,
		Status:    string(status),
	})
	if err != nil {
		return nil, err
	}
	d := mapDBMeetingParticipantToDomain(row)
	return &d, nil
}

func (r *meetingRepository) UpdateParticipantStatus(ctx context.Context, meetingID, userID string, status domain.ParticipantStatus) error {
	mid, err := uuid.Parse(meetingID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.UpdateParticipantStatus(ctx, db.UpdateParticipantStatusParams{
		MeetingID: mid,
		UserID:    uid,
		Status:    string(status),
	})
}

func (r *meetingRepository) GetMeetingParticipants(ctx context.Context, meetingID string) ([]domain.MeetingParticipant, error) {
	mid, err := uuid.Parse(meetingID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetMeetingParticipants(ctx, mid)
	if err != nil {
		return nil, err
	}
	result := make([]domain.MeetingParticipant, len(rows))
	for i, p := range rows {
		result[i] = mapDBMeetingParticipantToDomain(p)
	}
	return result, nil
}

func (r *meetingRepository) CreateReminder(_ context.Context, _, _ string, _ domain.ChannelType, _ time.Time) error {
	// meeting_reminders table removed in schema redesign
	return nil
}

func mapDBMeetingToDomain(m db.Meeting) domain.Meeting {
	var projectID *string
	if m.ProjectID.Valid {
		id := m.ProjectID.UUID.String()
		projectID = &id
	}
	var desc *string
	if m.Description.Valid {
		desc = &m.Description.String
	}
	var loc *string
	if m.Location.Valid {
		loc = &m.Location.String
	}
	return domain.Meeting{
		ID:          m.ID.String(),
		ProjectID:   projectID,
		Name:        m.Name,
		Description: desc,
		Type:        domain.MeetingType(m.MeetingType),
		Location:    loc,
		Status:      domain.MeetingStatus(m.Status),
		StartTime:   m.StartTime,
		EndTime:     m.EndTime,
		CreatedBy:   m.CreatedBy.String(),
	}
}

func mapDBMeetingParticipantToDomain(p db.MeetingParticipant) domain.MeetingParticipant {
	return domain.MeetingParticipant{
		ID:        p.ID.String(),
		MeetingID: p.MeetingID.String(),
		UserID:    p.UserID.String(),
		Status:    domain.ParticipantStatus(p.Status),
	}
}

