create table project(
    id bigserial PRIMARY KEY,
    key text,
    name text,
    type text,
    description text
);

create table author(
    id bigserial PRIMARY KEY,
    account_id text,
    display_name text,
    email text,
    active boolean
);

create table issues(
    id bigserial PRIMARY KEY,
    key text,
    project_id bigint,
    summary text,
    description text,
    issuetype text,
    status text,
    priority text,
    reporter_id bigint,
    created timestamp,
    updated timestamp,
    resolution text,
    duedate timestamp,
    labels text[],
    components text[],
    fix_versions text[],
    affected_versions text[],
    original_estimate int,
    time_spent int,
    remaining_estimate int,
    FOREIGN KEY (project_id) references project(id),
    Foreign Key (reporter_id) references author(id)
);

create table status_change(
    id bigserial PRIMARY KEY,
    issue_id bigint,
    author_id bigint,
    created timestamp,
    field text,
    from_value text,
    from_string text,
    to_value text,
    to_string text,
    FOREIGN KEY (issue_id) references issues(id),
    FOREIGN KEY (author_id) references author(id)
);