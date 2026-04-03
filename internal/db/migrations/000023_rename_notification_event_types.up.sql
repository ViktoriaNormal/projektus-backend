-- Rename notification event types to match frontend expectations.
UPDATE notifications SET event_type = 'comment_mention' WHERE event_type = 'task_mentioned_in_comment';
UPDATE notifications SET event_type = 'task_status_change_author' WHERE event_type = 'task_status_changed_author';
UPDATE notifications SET event_type = 'task_status_change_assignee' WHERE event_type = 'task_status_changed_assignee';
UPDATE notifications SET event_type = 'task_status_change_watcher' WHERE event_type = 'task_status_changed_watcher';
UPDATE notifications SET event_type = 'meeting_invite' WHERE event_type = 'meeting_invitation_received';
UPDATE notifications SET event_type = 'meeting_change' WHERE event_type = 'meeting_updated';
UPDATE notifications SET event_type = 'meeting_cancel' WHERE event_type = 'meeting_cancelled';

UPDATE notification_settings SET event_type = 'comment_mention' WHERE event_type = 'task_mentioned_in_comment';
UPDATE notification_settings SET event_type = 'task_status_change_author' WHERE event_type = 'task_status_changed_author';
UPDATE notification_settings SET event_type = 'task_status_change_assignee' WHERE event_type = 'task_status_changed_assignee';
UPDATE notification_settings SET event_type = 'task_status_change_watcher' WHERE event_type = 'task_status_changed_watcher';
UPDATE notification_settings SET event_type = 'meeting_invite' WHERE event_type = 'meeting_invitation_received';
UPDATE notification_settings SET event_type = 'meeting_change' WHERE event_type = 'meeting_updated';
UPDATE notification_settings SET event_type = 'meeting_cancel' WHERE event_type = 'meeting_cancelled';
