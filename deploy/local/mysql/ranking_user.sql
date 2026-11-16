CREATE USER 'ranking_app'@'%' IDENTIFIED BY 'dev-local-ranking';
GRANT SELECT,INSERT,UPDATE,DELETE ON activity_ranking.* TO 'ranking_app'@'%';
