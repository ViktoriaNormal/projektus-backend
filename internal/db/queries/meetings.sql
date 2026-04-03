-- Meetings

-- name: CreateMeeting :one
INSERT INTO meetings (project_id, name, description, meeting_type, location, start_time, end_time, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, project_id, name, description, meeting_type, start_time, end_time, created_by, location, status;

-- name: GetMeetingByID :one
SELECT id, project_id, name, description, meeting_type, start_time, end_time, created_by, location, status
FROM meetings
WHERE id = $1;

-- name: UpdateMeeting :exec
UPDATE meetings
SET name = $2,
    description = $3,
    meeting_type = $4,
    location = $5,
    start_time = $6,
    end_time = $7,
    project_id = $8
WHERE id = $1;

-- name: CancelMeeting :one
UPDATE meetings
SET status = 'cancelled'
WHERE id = $1
RETURNING id, project_id, name, description, meeting_type, start_time, end_time, created_by, location, status;

-- name: DeleteMeeting :exec
DELETE FROM meetings
WHERE id = $1;

-- name: ListUserMeetings :many
SELECT m.id, m.project_id, m.name, m.description, m.meeting_type, m.start_time, m.end_time, m.created_by, m.location, m.status
FROM meetings m
JOIN meeting_participants mp ON mp.meeting_id = m.id
WHERE mp.user_id = $1
  AND mp.status = 'accepted'
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR m.start_time >= sqlc.narg(from_time))
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR m.start_time <= sqlc.narg(to_time))
ORDER BY m.start_time;

-- name: ListProjectMeetings :many
SELECT id, project_id, name, description, meeting_type, start_time, end_time, created_by, location, status
FROM meetings
WHERE project_id = $1
ORDER BY start_time;

-- Participants

-- name: AddMeetingParticipant :one
INSERT INTO meeting_participants (meeting_id, user_id, status)
VALUES ($1, $2, $3)
ON CONFLICT (meeting_id, user_id) DO UPDATE
SET status = EXCLUDED.status
RETURNING id, meeting_id, user_id, status;

-- name: UpdateParticipantStatus :exec
UPDATE meeting_participants
SET status = $3
WHERE meeting_id = $1 AND user_id = $2;

-- name: GetMeetingParticipants :many
SELECT id, meeting_id, user_id, status
FROM meeting_participants
WHERE meeting_id = $1;

-- name: GetParticipantStatus :one
SELECT status
FROM meeting_participants
WHERE meeting_id = $1 AND user_id = $2;
