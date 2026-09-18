-- 1. Insert Users
INSERT INTO users (username, email) VALUES
('bugs_admin', 'admin@bugsbunny.local'),
('daffy_dev', 'daffy@bugsbunny.local'),
('lola_qa', 'lola@bugsbunny.local');

-- 2. Insert a Project
INSERT INTO projects (name, key, settings) VALUES
('Bugsbunny Core Engine', 'BUNNY', '{"public_access": true, "allow_guests": false}');

-- 3. Insert Workflow States
INSERT INTO workflow_states (project_id, name, category)
SELECT id, 'Triage', 'todo' FROM projects WHERE key = 'BUNNY' UNION ALL
SELECT id, 'In Progress', 'in_progress' FROM projects WHERE key = 'BUNNY' UNION ALL
SELECT id, 'In QA', 'in_progress' FROM projects WHERE key = 'BUNNY' UNION ALL
SELECT id, 'Resolved', 'done' FROM projects WHERE key = 'BUNNY';

-- 4. Insert Issues (Notice the custom_data JSON payloads!)
INSERT INTO issues (project_id, issue_key, title, body, state_id, reporter_id, assignee_id, custom_data)
VALUES
(
    (SELECT id FROM projects WHERE key = 'BUNNY'),
    'BUNNY-1',
    'Design WASM plugin architecture',
    'We need to finalize how community plugins will interface with the core engine.',
    (SELECT id FROM workflow_states WHERE name = 'In Progress' AND project_id = (SELECT id FROM projects WHERE key = 'BUNNY')),
    (SELECT id FROM users WHERE username = 'bugs_admin'),
    (SELECT id FROM users WHERE username = 'daffy_dev'),
    '{"priority": "critical", "component": "backend", "labels": ["architecture", "wasm"]}'
),
(
    (SELECT id FROM projects WHERE key = 'BUNNY'),
    'BUNNY-2',
    'Dark mode toggle missing on dashboard',
    'The dark mode toggle is completely hidden on Safari.',
    (SELECT id FROM workflow_states WHERE name = 'Triage' AND project_id = (SELECT id FROM projects WHERE key = 'BUNNY')),
    (SELECT id FROM users WHERE username = 'lola_qa'),
    NULL,
    '{"priority": "medium", "component": "frontend", "browser": "Safari", "os": "macOS"}'
);

-- 5. Insert Activity Stream (Audit Log)
INSERT INTO activity_stream (issue_id, actor_id, action_type, payload)
VALUES
(
    (SELECT id FROM issues WHERE issue_key = 'BUNNY-1'),
    (SELECT id FROM users WHERE username = 'daffy_dev'),
    'status_changed',
    '{"from": "Triage", "to": "In Progress"}'
),
(
    (SELECT id FROM issues WHERE issue_key = 'BUNNY-2'),
    (SELECT id FROM users WHERE username = 'lola_qa'),
    'commented',
    '{"text": "Confirmed this is still broken in version 1.0.2"}'
);
