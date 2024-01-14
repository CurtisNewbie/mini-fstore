alter table mini_fstore.file add column link varchar(32) NOT NULL default '' COMMENT 'symbolic link to another file id';
alter table mini_fstore.file add index link_idx (link);
alter table mini_fstore.file add index md5_size_name_idx (md5, size, name);
