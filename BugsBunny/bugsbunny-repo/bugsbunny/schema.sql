-- 1. Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Projects Table
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    key VARCHAR(10) UNIQUE NOT NULL,
    settings JSONB DEFAULT '{}'
);

-- 3. Workflow States Table
CREATE TABLE workflow_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    category VARCHAR(20) NOT NULL
);

-- 4. Issues Table (The Engine Core)
CREATE TABLE issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    issue_key VARCHAR(20) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    state_id UUID REFERENCES workflow_states(id),
    reporter_id UUID REFERENCES users(id),
    assignee_id UUID REFERENCES users(id),
    custom_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create a GIN index on custom_data for fast JSON queries
CREATE INDEX idx_issues_custom_data ON issues USING GIN (custom_data);

-- 5. Activity Stream Table
CREATE TABLE activity_stream (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id),
    action_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
