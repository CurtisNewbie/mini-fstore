CREATE DATABASE IF NOT EXISTS mini_fstore;

CREATE TABLE IF NOT EXISTS `file` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'primary key',
  `file_id` varchar(32) NOT NULL COMMENT 'file id',
  `name` varchar(255) NOT NULL COMMENT 'file name',
  `status` varchar(10) NOT NULL COMMENT 'status',
  `size` bigint NOT NULL COMMENT 'size in bytes',
  `md5` varchar(32) NOT NULL COMMENT 'md5',
  `upl_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'upload time',
  `log_del_time` timestamp NULL DEFAULT NULL COMMENT 'logic delete time',
  `phy_del_time` timestamp NULL DEFAULT NULL COMMENT 'physic delete time',
  PRIMARY KEY (`id`),
  KEY `file_id` (`file_id`,`status`)
) ENGINE=InnoDB COMMENT='File';
