UPDATE notifications SET event_type = 'task_mentioned_in_comment' WHERE event_type = 'comment_mention';
UPDATE notifications SET event_type = 'task_status_changed_author' WHERE event_type = 'task_status_change_author';
UPDATE notifications SET event_type = 'task_status_changed_assignee' WHERE event_type = 'task_status_change_assignee';
UPDATE notifications SET event_type = 'task_status_changed_watcher' WHERE event_type = 'task_status_change_watcher';
UPDATE notifications SET event_type = 'meeting_invitation_received' WHERE event_type = 'meeting_invite';
UPDATE notifications SET event_type = 'meeting_updated' WHERE event_type = 'meeting_change';
UPDATE notifications SET event_type = 'meeting_cancelled' WHERE event_type = 'meeting_cancel';

UPDATE notification_settings SET event_type = 'task_mentioned_in_comment' WHERE event_type = 'comment_mention';
UPDATE notification_settings SET event_type = 'task_status_changed_author' WHERE event_type = 'task_status_change_author';
UPDATE notification_settings SET event_type = 'task_status_changed_assignee' WHERE event_type = 'task_status_change_assignee';
UPDATE notification_settings SET event_type = 'task_status_changed_watcher' WHERE event_type = 'task_status_change_watcher';
UPDATE notification_settings SET event_type = 'meeting_invitation_received' WHERE event_type = 'meeting_invite';
UPDATE notification_settings SET event_type = 'meeting_updated' WHERE event_type = 'meeting_change';
UPDATE notification_settings SET event_type = 'meeting_cancelled' WHERE event_type = 'meeting_cancel';
