-- ============================================================
-- MCP Agent Server — Oracle 23ai Schema
-- 11 tables owned by MCP_USER in FREEPDB1.
-- Executed by SYS via gvenzl/oracle-free init scripts.
-- Using schema-qualified DDL so objects are owned by MCP_USER.
-- ============================================================

ALTER SESSION SET CONTAINER = FREEPDB1;


-- ============================================================
-- PROJECTS
-- ============================================================
CREATE TABLE MCP_USER.projects (
    id                   RAW(16)                      DEFAULT SYS_GUID() NOT NULL,
    api_key              VARCHAR2(64)                 NOT NULL,
    name                 VARCHAR2(255)                NOT NULL,
    created_at           TIMESTAMP WITH TIME ZONE     DEFAULT SYSTIMESTAMP NOT NULL,
    setup_completed_at   TIMESTAMP WITH TIME ZONE,
    CONSTRAINT pk_projects         PRIMARY KEY (id),
    CONSTRAINT uk_projects_api_key UNIQUE (api_key),
    CONSTRAINT uk_projects_name    UNIQUE (name)
);


-- ============================================================
-- USERS
-- ============================================================
CREATE TABLE MCP_USER.users (
    git_username  VARCHAR2(255)              NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE   DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_users PRIMARY KEY (git_username)
);


-- ============================================================
-- SESSIONS
-- ============================================================
CREATE TABLE MCP_USER.sessions (
    session_id     VARCHAR2(64)               NOT NULL,
    project_id     RAW(16)                    NOT NULL,
    git_username   VARCHAR2(255)              NOT NULL,
    agent          VARCHAR2(20)               NOT NULL,
    started_at     TIMESTAMP WITH TIME ZONE   DEFAULT SYSTIMESTAMP NOT NULL,
    last_activity  TIMESTAMP WITH TIME ZONE   DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_sessions             PRIMARY KEY (session_id),
    CONSTRAINT fk_sessions_project     FOREIGN KEY (project_id)   REFERENCES MCP_USER.projects (id),
    CONSTRAINT fk_sessions_user        FOREIGN KEY (git_username) REFERENCES MCP_USER.users (git_username),
    CONSTRAINT ck_sessions_agent       CHECK (agent IN ('claude', 'kiro'))
);


-- ============================================================
-- SKILLS
-- ============================================================
CREATE TABLE MCP_USER.skills (
    id         RAW(16)                      DEFAULT SYS_GUID() NOT NULL,
    name       VARCHAR2(255)                NOT NULL,
    content    CLOB                         NOT NULL,
    embedding  VECTOR(384, FLOAT32)        ,
    active     NUMBER(1)                    DEFAULT 1 NOT NULL,
    synced_at  TIMESTAMP WITH TIME ZONE     DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_skills        PRIMARY KEY (id),
    CONSTRAINT uk_skills_name   UNIQUE (name),
    CONSTRAINT ck_skills_active CHECK (active IN (0, 1))
);


-- ============================================================
-- SKILL_CHUNKS
-- ============================================================
CREATE TABLE MCP_USER.skill_chunks (
    id          RAW(16)                      DEFAULT SYS_GUID() NOT NULL,
    skill_id    RAW(16)                      NOT NULL,
    chunk_name  VARCHAR2(255)                NOT NULL,
    content     CLOB                         NOT NULL,
    embedding   VECTOR(384, FLOAT32)        ,
    position    NUMBER(5)                    NOT NULL,
    synced_at   TIMESTAMP WITH TIME ZONE     DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_skill_chunks       PRIMARY KEY (id),
    CONSTRAINT fk_skill_chunks_skill FOREIGN KEY (skill_id) REFERENCES MCP_USER.skills (id),
    CONSTRAINT uk_skill_chunks_name  UNIQUE (skill_id, chunk_name)
);


-- ============================================================
-- PROJECT_SKILLS
-- ============================================================
CREATE TABLE MCP_USER.project_skills (
    project_id  RAW(16)                      NOT NULL,
    skill_id    RAW(16)                      NOT NULL,
    active      NUMBER(1)                    DEFAULT 1 NOT NULL,
    enabled_at  TIMESTAMP WITH TIME ZONE     DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_project_skills         PRIMARY KEY (project_id, skill_id),
    CONSTRAINT fk_project_skills_project FOREIGN KEY (project_id) REFERENCES MCP_USER.projects (id),
    CONSTRAINT fk_project_skills_skill   FOREIGN KEY (skill_id)   REFERENCES MCP_USER.skills (id),
    CONSTRAINT ck_project_skills_active  CHECK (active IN (0, 1))
);


-- ============================================================
-- USER_SKILL_PREFS
-- ============================================================
CREATE TABLE MCP_USER.user_skill_prefs (
    git_username  VARCHAR2(255)              NOT NULL,
    project_id    RAW(16)                    NOT NULL,
    skill_id      RAW(16)                    NOT NULL,
    added_at      TIMESTAMP WITH TIME ZONE   DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_user_skill_prefs         PRIMARY KEY (git_username, project_id, skill_id),
    CONSTRAINT fk_user_skill_prefs_user    FOREIGN KEY (git_username) REFERENCES MCP_USER.users (git_username),
    CONSTRAINT fk_user_skill_prefs_project FOREIGN KEY (project_id)   REFERENCES MCP_USER.projects (id),
    CONSTRAINT fk_user_skill_prefs_skill   FOREIGN KEY (skill_id)     REFERENCES MCP_USER.skills (id)
);


-- ============================================================
-- MEMORY_CHANGES
-- ============================================================
CREATE TABLE MCP_USER.memory_changes (
    id              RAW(16)                  DEFAULT SYS_GUID() NOT NULL,
    project_id      RAW(16)                  NOT NULL,
    commit_hash     VARCHAR2(64)             NOT NULL,
    branch          VARCHAR2(255)            NOT NULL,
    author          VARCHAR2(255)            NOT NULL,
    file_path       VARCHAR2(1000)           NOT NULL,
    kind            VARCHAR2(10)            ,
    intent          VARCHAR2(50)            ,
    what            CLOB                     NOT NULL,
    why             CLOB                    ,
    language        VARCHAR2(50)            ,
    tags            VARCHAR2(500)           ,
    raw_diff        CLOB                    ,
    content_before  CLOB                    ,
    content_after   CLOB                    ,
    embedding       VECTOR(384, FLOAT32)    ,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_memory_changes             PRIMARY KEY (id),
    CONSTRAINT fk_memory_changes_project     FOREIGN KEY (project_id) REFERENCES MCP_USER.projects (id),
    CONSTRAINT uk_memory_changes_commit_file UNIQUE (project_id, commit_hash, file_path),
    CONSTRAINT ck_memory_changes_kind        CHECK (kind IN ('code', 'doc', 'config'))
);


-- ============================================================
-- MEMORY_CHANGE_HUNKS
-- ============================================================
CREATE TABLE MCP_USER.memory_change_hunks (
    id                RAW(16)                DEFAULT SYS_GUID() NOT NULL,
    memory_change_id  RAW(16)                NOT NULL,
    lines_start       NUMBER(10)             NOT NULL,
    lines_end         NUMBER(10)             NOT NULL,
    symbol            VARCHAR2(500)         ,
    change_type       VARCHAR2(20)           NOT NULL,
    hunk_diff         CLOB                   NOT NULL,
    CONSTRAINT pk_memory_change_hunks    PRIMARY KEY (id),
    CONSTRAINT fk_memory_change_hunks_mc
        FOREIGN KEY (memory_change_id) REFERENCES MCP_USER.memory_changes (id) ON DELETE CASCADE,
    CONSTRAINT ck_memory_change_hunks_type
        CHECK (change_type IN ('addition', 'deletion', 'modification'))
);


-- ============================================================
-- DOCUMENTS
-- ============================================================
CREATE TABLE MCP_USER.documents (
    id                  RAW(16)              DEFAULT SYS_GUID() NOT NULL,
    project_id          RAW(16)              NOT NULL,
    source_path         VARCHAR2(1000)       NOT NULL,
    title               VARCHAR2(500)        NOT NULL,
    doc_type            VARCHAR2(50)         NOT NULL,
    content             CLOB                 NOT NULL,
    embedding           VECTOR(384, FLOAT32),
    indexed_at          TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    source_modified_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    stale               NUMBER(1)            DEFAULT 0 NOT NULL,
    CONSTRAINT pk_documents          PRIMARY KEY (id),
    CONSTRAINT fk_documents_project  FOREIGN KEY (project_id) REFERENCES MCP_USER.projects (id),
    CONSTRAINT uk_documents_path     UNIQUE (project_id, source_path),
    CONSTRAINT ck_documents_stale    CHECK (stale IN (0, 1)),
    CONSTRAINT ck_documents_type     CHECK (doc_type IN (
        'ADR', 'API_SPEC', 'RUNBOOK', 'GUIDE', 'README',
        'CHANGELOG', 'ONBOARDING', 'DESIGN', 'OTHER'
    ))
);


-- ============================================================
-- ENRICHMENT_QUEUE
-- ============================================================
CREATE TABLE MCP_USER.enrichment_queue (
    id                RAW(16)                  DEFAULT SYS_GUID() NOT NULL,
    memory_change_id  RAW(16)                  NOT NULL,
    status            VARCHAR2(20)             DEFAULT 'PENDING' NOT NULL,
    attempts          NUMBER(3)                DEFAULT 0 NOT NULL,
    last_error        VARCHAR2(1000)          ,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    processed_at      TIMESTAMP WITH TIME ZONE,
    CONSTRAINT pk_enrichment_queue      PRIMARY KEY (id),
    CONSTRAINT fk_enrichment_queue_mc   FOREIGN KEY (memory_change_id) REFERENCES MCP_USER.memory_changes (id) ON DELETE CASCADE,
    CONSTRAINT uk_enrichment_queue_mc   UNIQUE (memory_change_id),
    CONSTRAINT ck_enrichment_status     CHECK (status IN ('PENDING', 'PROCESSING', 'DONE', 'FAILED'))
);

CREATE INDEX MCP_USER.idx_enrichment_queue_status ON MCP_USER.enrichment_queue (status, created_at);

COMMENT ON TABLE  MCP_USER.enrichment_queue              IS 'Persistent queue for commits needing LLM enrichment of intent/what/why fields.';
COMMENT ON COLUMN MCP_USER.enrichment_queue.status       IS 'PENDING → PROCESSING → DONE|FAILED. Uses SELECT FOR UPDATE SKIP LOCKED for concurrency.';
COMMENT ON COLUMN MCP_USER.enrichment_queue.attempts     IS 'Retry counter. Tasks with attempts >= 3 are marked FAILED.';


-- ============================================================
-- DOCUMENT_SECTIONS
-- ============================================================
CREATE TABLE MCP_USER.document_sections (
    id           RAW(16)                     DEFAULT SYS_GUID() NOT NULL,
    document_id  RAW(16)                     NOT NULL,
    heading      VARCHAR2(500)               NOT NULL,
    content      CLOB                        NOT NULL,
    embedding    VECTOR(384, FLOAT32)       ,
    position     NUMBER(5)                   NOT NULL,
    indexed_at   TIMESTAMP WITH TIME ZONE    DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_document_sections     PRIMARY KEY (id),
    CONSTRAINT fk_doc_sections_document
        FOREIGN KEY (document_id) REFERENCES MCP_USER.documents (id) ON DELETE CASCADE
);


-- ============================================================
-- VECTOR INDEXES  —  HNSW cosine similarity (Oracle 23ai)
-- ============================================================
CREATE VECTOR INDEX MCP_USER.vidx_skills_embedding
    ON MCP_USER.skills (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;

CREATE VECTOR INDEX MCP_USER.vidx_skill_chunks_embedding
    ON MCP_USER.skill_chunks (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;

CREATE VECTOR INDEX MCP_USER.vidx_memory_changes_embedding
    ON MCP_USER.memory_changes (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;

CREATE VECTOR INDEX MCP_USER.vidx_documents_embedding
    ON MCP_USER.documents (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;

CREATE VECTOR INDEX MCP_USER.vidx_document_sections_embedding
    ON MCP_USER.document_sections (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;


-- ============================================================
-- INDEXES
-- ============================================================
CREATE INDEX MCP_USER.idx_sessions_project  ON MCP_USER.sessions (project_id, started_at DESC);
CREATE INDEX MCP_USER.idx_sessions_user     ON MCP_USER.sessions (git_username, project_id);

CREATE INDEX MCP_USER.idx_skill_chunks_skill ON MCP_USER.skill_chunks (skill_id, position);

CREATE INDEX MCP_USER.idx_project_skills_skill ON MCP_USER.project_skills (skill_id);

CREATE INDEX MCP_USER.idx_user_skill_prefs_project ON MCP_USER.user_skill_prefs (project_id, git_username);

CREATE INDEX MCP_USER.idx_memory_changes_project ON MCP_USER.memory_changes (project_id, created_at DESC);
CREATE INDEX MCP_USER.idx_memory_changes_commit  ON MCP_USER.memory_changes (commit_hash);
CREATE INDEX MCP_USER.idx_memory_changes_file    ON MCP_USER.memory_changes (project_id, file_path);
CREATE INDEX MCP_USER.idx_memory_changes_kind    ON MCP_USER.memory_changes (project_id, kind);

CREATE INDEX MCP_USER.idx_memory_change_hunks_mc    ON MCP_USER.memory_change_hunks (memory_change_id);
CREATE INDEX MCP_USER.idx_memory_change_hunks_lines ON MCP_USER.memory_change_hunks (memory_change_id, lines_start, lines_end);

CREATE INDEX MCP_USER.idx_documents_project ON MCP_USER.documents (project_id, doc_type);
CREATE INDEX MCP_USER.idx_documents_stale   ON MCP_USER.documents (project_id, stale);

CREATE INDEX MCP_USER.idx_doc_sections_document ON MCP_USER.document_sections (document_id, position);


-- ============================================================
-- COMMENTS
-- ============================================================
COMMENT ON TABLE  MCP_USER.projects           IS 'Projects registered via REST (POST /api/projects). Identified by api_key in all requests.';
COMMENT ON COLUMN MCP_USER.projects.api_key              IS 'Secret key used to route all batch indexing and MCP calls to this project.';
COMMENT ON COLUMN MCP_USER.projects.name                 IS 'Human-readable project name, typically the git repository folder name.';
COMMENT ON COLUMN MCP_USER.projects.setup_completed_at   IS 'Timestamp of the last successful setupProject MCP tool call. NULL means setup has not been run yet.';

COMMENT ON TABLE  MCP_USER.users              IS 'Developers auto-created on first git activity. Identity from git config (user.name / user.email). No explicit registration.';
COMMENT ON COLUMN MCP_USER.users.git_username IS 'Primary key. Git author name — used as identity across sessions and commits.';

COMMENT ON TABLE  MCP_USER.sessions           IS 'One row per Claude or Kiro conversation. session_id sourced from hook payload.';
COMMENT ON COLUMN MCP_USER.sessions.agent     IS 'Agent type: claude or kiro.';

COMMENT ON TABLE  MCP_USER.skills             IS 'Global skill catalogue. Skills are not scoped to a project — any project can enable them via project_skills.';
COMMENT ON COLUMN MCP_USER.skills.embedding   IS '384-dim FLOAT32 vector (multilingual-e5-small). Used for cosine similarity search.';
COMMENT ON COLUMN MCP_USER.skills.synced_at   IS 'Last sync timestamp. Updated when content hash changes.';

COMMENT ON TABLE  MCP_USER.skill_chunks            IS 'Sub-documents of a skill with their own embedding for fine-grained RAG. Search returns the most relevant chunk; agent loads parent skill for full context.';
COMMENT ON COLUMN MCP_USER.skill_chunks.chunk_name IS 'Relative name/path of the sub-document within the skill. Unique within a skill.';
COMMENT ON COLUMN MCP_USER.skill_chunks.position   IS 'Order within the skill. Allows reconstructing the full skill by concatenating chunks in ascending order.';

COMMENT ON TABLE  MCP_USER.project_skills     IS 'Skills enabled for a project. Controlled by the project owner via MCP tools. active=0 disables without deleting.';

COMMENT ON TABLE  MCP_USER.user_skill_prefs   IS 'Per-user skill preferences within a project. Set via MCP tools during a session. Allows each developer to configure their own subset of enabled skills.';

COMMENT ON TABLE  MCP_USER.memory_changes                IS 'Vectorised git commit history. 1 row per file per commit. Enables semantic search over past decisions and implementations.';
COMMENT ON COLUMN MCP_USER.memory_changes.project_id     IS 'FK to PROJECTS.id. Scopes memory to a specific repository.';
COMMENT ON COLUMN MCP_USER.memory_changes.intent         IS 'Commit type: feat, fix, refactor, docs, test, chore, perf, style.';
COMMENT ON COLUMN MCP_USER.memory_changes.what           IS 'What changed. Parsed from commit body (what: line). Key field for semantic search.';
COMMENT ON COLUMN MCP_USER.memory_changes.why            IS 'Why this change was made. Parsed from commit body (why: line). Most important field for semantic search.';
COMMENT ON COLUMN MCP_USER.memory_changes.kind           IS 'File discriminator: code (source/test/script), doc (markdown/specs), config (build/infra). Drives queryCode and queryDocs MCP tools.';
COMMENT ON COLUMN MCP_USER.memory_changes.embedding      IS '384-dim vector of (intent + what + why + filePath). Enables intent-based similarity search.';

COMMENT ON TABLE  MCP_USER.memory_change_hunks               IS 'Individual @@ diff blocks of a file change. CASCADE DELETE from memory_changes. Enables granular line-range navigation.';
COMMENT ON COLUMN MCP_USER.memory_change_hunks.change_type   IS 'Hunk type: addition (+ only), deletion (- only), modification (both).';

COMMENT ON TABLE  MCP_USER.documents                    IS 'Project knowledge base: ADRs, API specs, runbooks, guides, READMEs, and other documentation indexed for semantic search.';
COMMENT ON COLUMN MCP_USER.documents.doc_type           IS 'Document type: ADR, API_SPEC, RUNBOOK, GUIDE, README, CHANGELOG, ONBOARDING, DESIGN, OTHER.';
COMMENT ON COLUMN MCP_USER.documents.stale              IS '1 = potentially outdated relative to recent changes, 0 = current.';

COMMENT ON TABLE  MCP_USER.document_sections            IS 'Individual sections of a document with their own embedding for granular RAG retrieval by heading.';
COMMENT ON COLUMN MCP_USER.document_sections.position   IS 'Section order within the document (0-based).';
