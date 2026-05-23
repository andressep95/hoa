-- ============================================================
-- HOA — Oracle 23ai Schema
-- 11 tables owned by HOA in FREEPDB1.
-- Executed by SYS via gvenzl/oracle-free init scripts.
-- Using schema-qualified DDL so objects are owned by HOA.
-- ============================================================

ALTER SESSION SET CONTAINER = FREEPDB1;


-- ============================================================
-- PROJECTS
-- ============================================================
CREATE TABLE HOA.projects (
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
CREATE TABLE HOA.users (
    git_username  VARCHAR2(255)              NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE   DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_users PRIMARY KEY (git_username)
);


-- ============================================================
-- SESSIONS
-- ============================================================
CREATE TABLE HOA.sessions (
    session_id     VARCHAR2(64)               NOT NULL,
    project_id     RAW(16)                    NOT NULL,
    git_username   VARCHAR2(255)              NOT NULL,
    agent          VARCHAR2(20)               NOT NULL,
    started_at     TIMESTAMP WITH TIME ZONE   DEFAULT SYSTIMESTAMP NOT NULL,
    last_activity  TIMESTAMP WITH TIME ZONE   DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_sessions             PRIMARY KEY (session_id),
    CONSTRAINT fk_sessions_project     FOREIGN KEY (project_id)   REFERENCES HOA.projects (id),
    CONSTRAINT fk_sessions_user        FOREIGN KEY (git_username) REFERENCES HOA.users (git_username),
    CONSTRAINT ck_sessions_agent       CHECK (agent IN ('claude', 'kiro', 'hoa'))
);


-- ============================================================
-- SKILLS
-- ============================================================
CREATE TABLE HOA.skills (
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
CREATE TABLE HOA.skill_chunks (
    id          RAW(16)                      DEFAULT SYS_GUID() NOT NULL,
    skill_id    RAW(16)                      NOT NULL,
    chunk_name  VARCHAR2(255)                NOT NULL,
    content     CLOB                         NOT NULL,
    embedding   VECTOR(384, FLOAT32)        ,
    position    NUMBER(5)                    NOT NULL,
    synced_at   TIMESTAMP WITH TIME ZONE     DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_skill_chunks       PRIMARY KEY (id),
    CONSTRAINT fk_skill_chunks_skill FOREIGN KEY (skill_id) REFERENCES HOA.skills (id),
    CONSTRAINT uk_skill_chunks_name  UNIQUE (skill_id, chunk_name)
);


-- ============================================================
-- PROJECT_SKILLS
-- ============================================================
CREATE TABLE HOA.project_skills (
    project_id  RAW(16)                      NOT NULL,
    skill_id    RAW(16)                      NOT NULL,
    active      NUMBER(1)                    DEFAULT 1 NOT NULL,
    enabled_at  TIMESTAMP WITH TIME ZONE     DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_project_skills         PRIMARY KEY (project_id, skill_id),
    CONSTRAINT fk_project_skills_project FOREIGN KEY (project_id) REFERENCES HOA.projects (id),
    CONSTRAINT fk_project_skills_skill   FOREIGN KEY (skill_id)   REFERENCES HOA.skills (id),
    CONSTRAINT ck_project_skills_active  CHECK (active IN (0, 1))
);


-- ============================================================
-- USER_SKILL_PREFS
-- ============================================================
CREATE TABLE HOA.user_skill_prefs (
    git_username  VARCHAR2(255)              NOT NULL,
    project_id    RAW(16)                    NOT NULL,
    skill_id      RAW(16)                    NOT NULL,
    added_at      TIMESTAMP WITH TIME ZONE   DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_user_skill_prefs         PRIMARY KEY (git_username, project_id, skill_id),
    CONSTRAINT fk_user_skill_prefs_user    FOREIGN KEY (git_username) REFERENCES HOA.users (git_username),
    CONSTRAINT fk_user_skill_prefs_project FOREIGN KEY (project_id)   REFERENCES HOA.projects (id),
    CONSTRAINT fk_user_skill_prefs_skill   FOREIGN KEY (skill_id)     REFERENCES HOA.skills (id)
);


-- ============================================================
-- MEMORY_CHANGES
-- ============================================================
CREATE TABLE HOA.memory_changes (
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
    CONSTRAINT fk_memory_changes_project     FOREIGN KEY (project_id) REFERENCES HOA.projects (id),
    CONSTRAINT uk_memory_changes_commit_file UNIQUE (project_id, commit_hash, file_path),
    CONSTRAINT ck_memory_changes_kind        CHECK (kind IN ('code', 'doc', 'config'))
);


-- ============================================================
-- MEMORY_CHANGE_HUNKS
-- ============================================================
CREATE TABLE HOA.memory_change_hunks (
    id                RAW(16)                DEFAULT SYS_GUID() NOT NULL,
    memory_change_id  RAW(16)                NOT NULL,
    lines_start       NUMBER(10)             NOT NULL,
    lines_end         NUMBER(10)             NOT NULL,
    symbol            VARCHAR2(500)         ,
    change_type       VARCHAR2(20)           NOT NULL,
    hunk_diff         CLOB                   NOT NULL,
    CONSTRAINT pk_memory_change_hunks    PRIMARY KEY (id),
    CONSTRAINT fk_memory_change_hunks_mc
        FOREIGN KEY (memory_change_id) REFERENCES HOA.memory_changes (id) ON DELETE CASCADE,
    CONSTRAINT ck_memory_change_hunks_type
        CHECK (change_type IN ('addition', 'deletion', 'modification'))
);


-- ============================================================
-- DOCUMENTS
-- ============================================================
CREATE TABLE HOA.documents (
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
    CONSTRAINT fk_documents_project  FOREIGN KEY (project_id) REFERENCES HOA.projects (id),
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
CREATE TABLE HOA.enrichment_queue (
    id                RAW(16)                  DEFAULT SYS_GUID() NOT NULL,
    memory_change_id  RAW(16)                  NOT NULL,
    status            VARCHAR2(20)             DEFAULT 'PENDING' NOT NULL,
    attempts          NUMBER(3)                DEFAULT 0 NOT NULL,
    last_error        VARCHAR2(1000)          ,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    processed_at      TIMESTAMP WITH TIME ZONE,
    CONSTRAINT pk_enrichment_queue      PRIMARY KEY (id),
    CONSTRAINT fk_enrichment_queue_mc   FOREIGN KEY (memory_change_id) REFERENCES HOA.memory_changes (id) ON DELETE CASCADE,
    CONSTRAINT uk_enrichment_queue_mc   UNIQUE (memory_change_id),
    CONSTRAINT ck_enrichment_status     CHECK (status IN ('PENDING', 'PROCESSING', 'DONE', 'FAILED'))
);

CREATE INDEX HOA.idx_enrichment_queue_status ON HOA.enrichment_queue (status, created_at);

COMMENT ON TABLE  HOA.enrichment_queue              IS 'Persistent queue for commits needing LLM enrichment of intent/what/why fields.';
COMMENT ON COLUMN HOA.enrichment_queue.status       IS 'PENDING → PROCESSING → DONE|FAILED. Uses SELECT FOR UPDATE SKIP LOCKED for concurrency.';
COMMENT ON COLUMN HOA.enrichment_queue.attempts     IS 'Retry counter. Tasks with attempts >= 3 are marked FAILED.';


-- ============================================================
-- DOCUMENT_SECTIONS
-- ============================================================
CREATE TABLE HOA.document_sections (
    id           RAW(16)                     DEFAULT SYS_GUID() NOT NULL,
    document_id  RAW(16)                     NOT NULL,
    heading      VARCHAR2(500)               NOT NULL,
    content      CLOB                        NOT NULL,
    embedding    VECTOR(384, FLOAT32)       ,
    position     NUMBER(5)                   NOT NULL,
    indexed_at   TIMESTAMP WITH TIME ZONE    DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT pk_document_sections     PRIMARY KEY (id),
    CONSTRAINT fk_doc_sections_document
        FOREIGN KEY (document_id) REFERENCES HOA.documents (id) ON DELETE CASCADE
);


-- ============================================================
-- VECTOR INDEXES  —  HNSW cosine similarity (Oracle 23ai)
-- ============================================================
CREATE VECTOR INDEX HOA.vidx_skills_embedding
    ON HOA.skills (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;

CREATE VECTOR INDEX HOA.vidx_skill_chunks_embedding
    ON HOA.skill_chunks (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;

CREATE VECTOR INDEX HOA.vidx_memory_changes_embedding
    ON HOA.memory_changes (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;

CREATE VECTOR INDEX HOA.vidx_documents_embedding
    ON HOA.documents (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;

CREATE VECTOR INDEX HOA.vidx_document_sections_embedding
    ON HOA.document_sections (embedding)
    ORGANIZATION NEIGHBOR PARTITIONS
    WITH DISTANCE COSINE
    WITH TARGET ACCURACY 95;


-- ============================================================
-- INDEXES
-- ============================================================
CREATE INDEX HOA.idx_sessions_project  ON HOA.sessions (project_id, started_at DESC);
CREATE INDEX HOA.idx_sessions_user     ON HOA.sessions (git_username, project_id);

CREATE INDEX HOA.idx_skill_chunks_skill ON HOA.skill_chunks (skill_id, position);

CREATE INDEX HOA.idx_project_skills_skill ON HOA.project_skills (skill_id);

CREATE INDEX HOA.idx_user_skill_prefs_project ON HOA.user_skill_prefs (project_id, git_username);

CREATE INDEX HOA.idx_memory_changes_project ON HOA.memory_changes (project_id, created_at DESC);
CREATE INDEX HOA.idx_memory_changes_commit  ON HOA.memory_changes (commit_hash);
CREATE INDEX HOA.idx_memory_changes_file    ON HOA.memory_changes (project_id, file_path);
CREATE INDEX HOA.idx_memory_changes_kind    ON HOA.memory_changes (project_id, kind);

CREATE INDEX HOA.idx_memory_change_hunks_mc    ON HOA.memory_change_hunks (memory_change_id);
CREATE INDEX HOA.idx_memory_change_hunks_lines ON HOA.memory_change_hunks (memory_change_id, lines_start, lines_end);

CREATE INDEX HOA.idx_documents_project ON HOA.documents (project_id, doc_type);
CREATE INDEX HOA.idx_documents_stale   ON HOA.documents (project_id, stale);

CREATE INDEX HOA.idx_doc_sections_document ON HOA.document_sections (document_id, position);


-- ============================================================
-- COMMENTS
-- ============================================================
COMMENT ON TABLE  HOA.projects           IS 'Projects registered by HOA. Identified by api_key.';
COMMENT ON COLUMN HOA.projects.api_key              IS 'Secret key used to route indexing calls to this project.';
COMMENT ON COLUMN HOA.projects.name                 IS 'Human-readable project name, typically the git repository folder name.';
COMMENT ON COLUMN HOA.projects.setup_completed_at   IS 'Timestamp of last successful project setup. NULL means not yet configured.';

COMMENT ON TABLE  HOA.users              IS 'Developers auto-created on first git activity. Identity from git config (user.name / user.email). No explicit registration.';
COMMENT ON COLUMN HOA.users.git_username IS 'Primary key. Git author name — used as identity across sessions and commits.';

COMMENT ON TABLE  HOA.sessions           IS 'One row per agent conversation session.';
COMMENT ON COLUMN HOA.sessions.agent     IS 'Agent type: claude, kiro, or hoa.';

COMMENT ON TABLE  HOA.skills             IS 'Global skill catalogue. Any project can enable them via project_skills.';
COMMENT ON COLUMN HOA.skills.embedding   IS '384-dim FLOAT32 vector (multilingual-e5-small). Used for cosine similarity search.';
COMMENT ON COLUMN HOA.skills.synced_at   IS 'Last sync timestamp. Updated when content hash changes.';

COMMENT ON TABLE  HOA.skill_chunks            IS 'Sub-documents of a skill with their own embedding for fine-grained RAG.';
COMMENT ON COLUMN HOA.skill_chunks.chunk_name IS 'Relative name/path of the sub-document within the skill.';
COMMENT ON COLUMN HOA.skill_chunks.position   IS 'Order within the skill.';

COMMENT ON TABLE  HOA.project_skills     IS 'Skills enabled for a project. active=0 disables without deleting.';

COMMENT ON TABLE  HOA.user_skill_prefs   IS 'Per-user skill preferences within a project.';

COMMENT ON TABLE  HOA.memory_changes                IS 'Vectorised git commit history. 1 row per file per commit.';
COMMENT ON COLUMN HOA.memory_changes.project_id     IS 'FK to PROJECTS.id. Scopes memory to a specific repository.';
COMMENT ON COLUMN HOA.memory_changes.intent         IS 'Commit type: feat, fix, refactor, docs, test, chore, perf, style.';
COMMENT ON COLUMN HOA.memory_changes.what           IS 'What changed. Key field for semantic search.';
COMMENT ON COLUMN HOA.memory_changes.why            IS 'Why this change was made. Most important field for semantic search.';
COMMENT ON COLUMN HOA.memory_changes.kind           IS 'File discriminator: code, doc, config.';
COMMENT ON COLUMN HOA.memory_changes.embedding      IS '384-dim vector of (intent + what + why + filePath).';

COMMENT ON TABLE  HOA.memory_change_hunks               IS 'Individual @@ diff blocks of a file change.';
COMMENT ON COLUMN HOA.memory_change_hunks.change_type   IS 'Hunk type: addition, deletion, modification.';

COMMENT ON TABLE  HOA.documents                    IS 'Project knowledge base: ADRs, API specs, runbooks, guides, READMEs, and other documentation indexed for semantic search.';
COMMENT ON COLUMN HOA.documents.doc_type           IS 'Document type: ADR, API_SPEC, RUNBOOK, GUIDE, README, CHANGELOG, ONBOARDING, DESIGN, OTHER.';
COMMENT ON COLUMN HOA.documents.stale              IS '1 = potentially outdated relative to recent changes, 0 = current.';

COMMENT ON TABLE  HOA.document_sections            IS 'Individual sections of a document with their own embedding for granular RAG retrieval by heading.';
COMMENT ON COLUMN HOA.document_sections.position   IS 'Section order within the document (0-based).';
