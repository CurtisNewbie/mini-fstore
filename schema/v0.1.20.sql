alter table mini_fstore.file add column `sha1` varchar(40) NOT NULL DEFAULT '' COMMENT 'sha1';
alter table mini_fstore.file add key sha1_size_idx (`sha1`,`size`);