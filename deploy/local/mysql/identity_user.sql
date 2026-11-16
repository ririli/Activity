CREATE USER 'identity_app'@'%' IDENTIFIED BY 'dev-local-identity';
GRANT SELECT,INSERT,UPDATE,DELETE ON activity_identity.* TO 'identity_app'@'%';
