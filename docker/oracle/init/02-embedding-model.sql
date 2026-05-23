-- ============================================================
-- HOA — Load ONNX embedding model + auto-embedding triggers
-- Runs after 01-schema.sql (alphabetical order in init dir)
-- Model must exist at /opt/oracle/models/all_MiniLM_L12_v2.onnx
-- Run docker/setup-model.sh before first docker compose up
-- ============================================================

ALTER SESSION SET CONTAINER = FREEPDB1;

-- ── Grant required privileges ───────────────────────────────────────────────

GRANT CREATE MINING MODEL TO hoa;
CREATE OR REPLACE DIRECTORY HOA_MODELS AS '/opt/oracle/models';
GRANT READ ON DIRECTORY HOA_MODELS TO hoa;

-- ── Load the ONNX model as HOA user ─────────────────────────────────────────

-- Connect as HOA to own the model
CONNECT hoa/Hoa_User23!@//localhost:1521/FREEPDB1

BEGIN
  BEGIN
    DBMS_VECTOR.DROP_ONNX_MODEL(model_name => 'HOA_EMBED_MODEL', force => true);
  EXCEPTION WHEN OTHERS THEN NULL;
  END;

  DBMS_VECTOR.LOAD_ONNX_MODEL(
    directory  => 'HOA_MODELS',
    file_name  => 'all_MiniLM_L12_v2.onnx',
    model_name => 'HOA_EMBED_MODEL',
    metadata   => JSON('{"function":"embedding","embeddingOutput":"embedding","input":{"input":["DATA"]}}')
  );
END;
/

-- ── Trigger: auto-generate embedding on memory_changes ──────────────────────

CREATE OR REPLACE TRIGGER TRG_MEMORY_CHANGES_EMBEDDING
BEFORE INSERT OR UPDATE ON memory_changes
FOR EACH ROW
DECLARE
  v_text VARCHAR2(4000);
BEGIN
  v_text := SUBSTR(
    NVL(:NEW.intent, '') || ' ' ||
    NVL(DBMS_LOB.SUBSTR(:NEW.what, 2000, 1), '') || ' ' ||
    NVL(DBMS_LOB.SUBSTR(:NEW.why, 1500, 1), '') || ' ' ||
    NVL(:NEW.file_path, ''),
    1, 4000
  );
  SELECT VECTOR_EMBEDDING(HOA_EMBED_MODEL USING v_text AS data)
    INTO :NEW.embedding
    FROM DUAL;
EXCEPTION
  WHEN OTHERS THEN NULL;
END;
/

-- ── Trigger: auto-generate embedding on skills ──────────────────────────────

CREATE OR REPLACE TRIGGER TRG_SKILLS_EMBEDDING
BEFORE INSERT OR UPDATE ON skills
FOR EACH ROW
DECLARE
  v_text VARCHAR2(4000);
BEGIN
  v_text := SUBSTR(:NEW.name || ' ' || NVL(DBMS_LOB.SUBSTR(:NEW.content, 3900, 1), ''), 1, 4000);
  SELECT VECTOR_EMBEDDING(HOA_EMBED_MODEL USING v_text AS data)
    INTO :NEW.embedding
    FROM DUAL;
EXCEPTION
  WHEN OTHERS THEN NULL;
END;
/

-- ── Trigger: auto-generate embedding on documents ───────────────────────────

CREATE OR REPLACE TRIGGER TRG_DOCUMENTS_EMBEDDING
BEFORE INSERT OR UPDATE ON documents
FOR EACH ROW
DECLARE
  v_text VARCHAR2(4000);
BEGIN
  v_text := SUBSTR(:NEW.title || ' ' || NVL(DBMS_LOB.SUBSTR(:NEW.content, 3900, 1), ''), 1, 4000);
  SELECT VECTOR_EMBEDDING(HOA_EMBED_MODEL USING v_text AS data)
    INTO :NEW.embedding
    FROM DUAL;
EXCEPTION
  WHEN OTHERS THEN NULL;
END;
/

-- ── Trigger: auto-generate embedding on document_sections ───────────────────

CREATE OR REPLACE TRIGGER TRG_DOC_SECTIONS_EMBEDDING
BEFORE INSERT OR UPDATE ON document_sections
FOR EACH ROW
DECLARE
  v_text VARCHAR2(4000);
BEGIN
  v_text := SUBSTR(:NEW.heading || ' ' || NVL(DBMS_LOB.SUBSTR(:NEW.content, 3900, 1), ''), 1, 4000);
  SELECT VECTOR_EMBEDDING(HOA_EMBED_MODEL USING v_text AS data)
    INTO :NEW.embedding
    FROM DUAL;
EXCEPTION
  WHEN OTHERS THEN NULL;
END;
/


-- ── Trigger: auto-generate embedding on feedback_rules ──────────────────────

CREATE OR REPLACE TRIGGER TRG_FEEDBACK_RULES_EMBEDDING
BEFORE INSERT OR UPDATE ON feedback_rules
FOR EACH ROW
DECLARE
  v_text VARCHAR2(4000);
BEGIN
  v_text := SUBSTR(:NEW.rule || ' ' || NVL(:NEW.why, '') || ' ' || NVL(:NEW.scope, ''), 1, 4000);
  SELECT VECTOR_EMBEDDING(HOA_EMBED_MODEL USING v_text AS data)
    INTO :NEW.embedding
    FROM DUAL;
EXCEPTION
  WHEN OTHERS THEN NULL;
END;
/