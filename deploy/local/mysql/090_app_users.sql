USE activity_identity;
CREATE USER 'identity_app'@'%' IDENTIFIED BY 'dev-local-identity';
GRANT SELECT,INSERT,UPDATE,DELETE ON activity_identity.* TO 'identity_app'@'%';
CREATE USER 'core_app'@'%' IDENTIFIED BY 'dev-local-core';
GRANT SELECT,INSERT,UPDATE,DELETE ON activity_core.* TO 'core_app'@'%';
CREATE USER 'ranking_app'@'%' IDENTIFIED BY 'dev-local-ranking';
GRANT SELECT,INSERT,UPDATE,DELETE ON activity_ranking.* TO 'ranking_app'@'%';
