-- ============================================================
-- 00-grants.sql — Switch to PDB and grant DDL privs to HOA
-- gvenzl/oracle-free runs init scripts as SYS in CDB (FREE).
-- HOA lives in FREEPDB1.
-- ============================================================

ALTER SESSION SET CONTAINER = FREEPDB1;

ALTER USER HOA QUOTA UNLIMITED ON USERS;
GRANT CREATE TABLE TO HOA;
GRANT CREATE SEQUENCE TO HOA;
GRANT CREATE PROCEDURE TO HOA;
