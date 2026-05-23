-- ============================================================
-- 00-grants.sql — Switch to PDB and grant DDL privs to MCP_USER
-- gvenzl/oracle-free runs init scripts as SYS in CDB (FREE).
-- MCP_USER lives in FREEPDB1.
-- ============================================================

ALTER SESSION SET CONTAINER = FREEPDB1;

ALTER USER MCP_USER QUOTA UNLIMITED ON USERS;
GRANT CREATE TABLE TO MCP_USER;
GRANT CREATE SEQUENCE TO MCP_USER;
GRANT CREATE PROCEDURE TO MCP_USER;
