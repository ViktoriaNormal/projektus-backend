-- Notification settings and feed, meetings and reminders

-- Notification settings per user and event type
CREATE TABLE notification_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    in_system BOOLEAN NOT NULL DEFAULT TRUE,
    in_email BOOLEAN NOT NULL DEFAULT FALSE,
    -- offset in minutes before event/deadline; NULL when not applicable
    reminder_offset_minutes INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_notification_settings_user_event UNIQUE (user_id, event_type)
);

CREATE INDEX idx_notification_settings_user_id ON notification_settings (user_id);

-- Notification feed (in-system + email status)
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'system', -- 'system' | 'email'
    title VARCHAR(255) NOT NULL,
    body TEXT,
    payload JSONB,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ,
    email_status VARCHAR(20), -- NULL | 'pending' | 'sent' | 'failed'
    email_sent_at TIMESTAMPTZ
);

CREATE INDEX idx_notifications_user_created_at ON notifications (user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id, is_read, created_at DESC);

-- Meetings (events)
CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    meeting_type VARCHAR(100) NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    canceled_at TIMESTAMPTZ
);

CREATE INDEX idx_meetings_created_by ON meetings (created_by);
CREATE INDEX idx_meetings_time ON meetings (start_time, end_time);

-- Meeting participants
CREATE TABLE meeting_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending' | 'accepted' | 'declined'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_meeting_participants_meeting_user UNIQUE (meeting_id, user_id)
);

CREATE INDEX idx_meeting_participants_user ON meeting_participants (user_id);

-- Meeting reminders (to avoid duplicate notifications)
CREATE TABLE meeting_reminders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL, -- 'system' | 'email'
    reminder_time TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    CONSTRAINT uq_meeting_reminders_meeting_user_channel_time UNIQUE (meeting_id, user_id, channel, reminder_time)
);

CREATE INDEX idx_meeting_reminders_user_time ON meeting_reminders (user_id, reminder_time);

