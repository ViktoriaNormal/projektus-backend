-- Meetings

-- name: CreateMeeting :one
INSERT INTO meetings (project_id, name, description, meeting_type, location, start_time, end_time, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetMeetingByID :one
SELECT *
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
    updated_at = NOW()
WHERE id = $1;

-- name: CancelMeeting :one
UPDATE meetings
SET status     = 'cancelled',
    canceled_at = NOW(),
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteMeeting :exec
DELETE FROM meetings
WHERE id = $1;

-- name: ListUserMeetings :many
SELECT m.*
FROM meetings m
JOIN meeting_participants mp ON mp.meeting_id = m.id
WHERE mp.user_id = $1
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR m.start_time >= sqlc.narg(from_time))
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR m.start_time <= sqlc.narg(to_time))
ORDER BY m.start_time;

-- name: ListProjectMeetings :many
SELECT *
FROM meetings
WHERE project_id = $1
ORDER BY start_time;

-- Participants

-- name: AddMeetingParticipant :one
INSERT INTO meeting_participants (meeting_id, user_id, status)
VALUES ($1, $2, $3)
ON CONFLICT (meeting_id, user_id) DO UPDATE
SET status = EXCLUDED.status,
    updated_at = NOW()
RETURNING *;

-- name: UpdateParticipantStatus :exec
UPDATE meeting_participants
SET status = $3,
    updated_at = NOW()
WHERE meeting_id = $1 AND user_id = $2;

-- name: GetMeetingParticipants :many
SELECT *
FROM meeting_participants
WHERE meeting_id = $1;

-- Reminders

-- name: CreateMeetingReminder :exec
INSERT INTO meeting_reminders (meeting_id, user_id, channel, reminder_time, sent_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (meeting_id, user_id, channel, reminder_time) DO NOTHING;

-- name: GetUpcomingMeetingsForUser :many
SELECT m.*, mr.channel, mr.reminder_time, mr.sent_at
FROM meetings m
JOIN meeting_participants mp ON mp.meeting_id = m.id
LEFT JOIN meeting_reminders mr
  ON mr.meeting_id = m.id
 AND mr.user_id = mp.user_id
WHERE mp.user_id = $1
  AND m.start_time BETWEEN $2 AND $3
  AND (mr.sent_at IS NULL)
ORDER BY m.start_time;

