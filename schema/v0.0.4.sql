alter table mini_fstore.file drop key md5;
alter table mini_fstore.file drop key file_id;
alter table mini_fstore.file add index (file_id, status);