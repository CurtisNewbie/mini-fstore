CREATE DATABASE IF NOT EXISTS mini_fstore;

CREATE TABLE mini_fstore.file (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT 'primary key',
  `file_id` varchar(32) NOT NULL COMMENT 'file id',
  `link` varchar(32) NOT NULL DEFAULT '' COMMENT 'symbolic link to another file id',
  `name` varchar(255) NOT NULL COMMENT 'file name',
  `status` varchar(10) NOT NULL COMMENT 'status',
  `size` bigint(20) NOT NULL COMMENT 'size in bytes',
  `md5` varchar(32) NOT NULL COMMENT 'md5',
  `upl_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'upload time',
  `log_del_time` timestamp NULL DEFAULT NULL COMMENT 'logic delete time',
  `phy_del_time` timestamp NULL DEFAULT NULL COMMENT 'physic delete time',
  `sha1` varchar(40) NOT NULL DEFAULT '' COMMENT 'sha1',
  PRIMARY KEY (`id`),
  KEY `file_id` (`file_id`,`status`),
  KEY `link_idx` (`link`),
  KEY `md5_size_name_idx` (`md5`,`size`,`name`),
  KEY `sha1_size_idx` (`sha1`,`size`)
) ENGINE=InnoDB COMMENT='File';
