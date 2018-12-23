CREATE DATABASE blue;

GRANT SELECT,INSERT,UPDATE,DELETE,CREATE,DROP,ALTER  ON blue.* to blue@localhost IDENTIFIED BY 'blue@123';

SET PASSWORD FOR 'blue'@'localhost' = PASSWORD('blue@123');

SET PASSWORD FOR 'blue'@'%' = PASSWORD('blue@123');

select host,user,password from user;