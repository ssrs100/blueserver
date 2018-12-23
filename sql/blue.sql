drop table if exists user;
create table user
(
  id      varchar(64)  not null,
  name    varchar(128) not null,
  passwd  varchar(128) not null,
  email   varchar(128),
  mobile  varchar(128),
  address varchar(512),
  primary key (id)
);

drop table if exists beacon;
create table beacon
(
  id          varchar(64)  not null,
  project_id  varchar(64)  not null,
  type        varchar(64)  not null,
  device_id   varchar(128) not null,
  status      varchar(32)  not null,
  create_at   datetime,
  description varchar(256),
  primary key (id)
);

drop table if exists attachment;
create table attachment
(
  id              varchar(64)  not null,
  beacon_id       varchar(64)  not null,
  attachment_name varchar(64)  not null,
  attachment_type varchar(128) not null,
  data            text         not null,
  create_at       datetime,
  primary key (id),
  FOREIGN KEY (`beacon_id`) REFERENCES beacon (`id`)
);

drop table if exists component;
create table component
(
  id                 varchar(64) not null,
  project_id         varchar(64) not null,
  name               varchar(64) not null,
  type               varchar(64) not null,
  mac_addr           varchar(64) not null,
  component_password varchar(64) not null,
  create_at          datetime,
  primary key (id)
);

drop table if exists component_detail;
create table component_detail
(
  id             varchar(64) not null,
  component_id   varchar(64) not null,
  component_name varchar(64) not null,
  adv_interval   int,
  tx_power       int,
  slot           int,
  update_status  int,
  data           text,
  update_data    text,
  primary key (id),
  FOREIGN KEY (`component_id`) REFERENCES component (`id`) ON DELETE CASCADE ON UPDATE CASCADE
);

drop table if exists collection;
create table collection
(
  id           varchar(64) not null,
  component_id varchar(64) not null,
  rssi         int,
  create_at    datetime,
  data         text,
  primary key (id),
  index        index_collection(component_id, create_at),
  FOREIGN KEY (`component_id`) REFERENCES component (`id`) ON DELETE CASCADE ON UPDATE CASCADE
);