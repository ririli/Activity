CREATE USER 'core_app'@'%' IDENTIFIED BY 'dev-local-core';
GRANT SELECT,INSERT,UPDATE,DELETE ON activity_core.* TO 'core_app'@'%';
