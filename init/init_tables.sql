ALTER DATABASE testdb set timezone to 'UTC';

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
    created timestamptz not null,
    updated timestamptz not null,
    resolution text,
    duedate timestamptz not null,
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
    created timestamptz not null ,
    field text,
    from_value text,
    from_string text,
    to_value text,
    to_string text,
    FOREIGN KEY (issue_id) references issues(id),
    FOREIGN KEY (author_id) references author(id)
);


create user pguser with password 'pgpwd';
grant connect on database testdb to pguser;
grant usage on schema public to pguser;
grant select, insert, update, delete on project, issues, author, status_change to pguser;
grant usage, select, update on all sequences in schema public to pguser;

alter default privileges in schema public
grant select, insert, delete, update on tables to pguser;

alter default privileges in schema public
grant usage, select, update on sequences to pguser;

create role replicator with replication login password 'postgres1';