create table config
(
    mail_server   tinytext not null,
    mail_username tinytext not null,
    mail_password tinytext not null,
    mail_port     tinytext not null,
    server_name   tinytext not null
);

create table files
(
    Id             int auto_increment
        primary key,
    DocName        text     not null,
    SenderName     text     not null,
    SenderEmail    text     not null,
    RecipientName  text     not null,
    RecipientEmail text     not null,
    Password       char(60) not null,
    CreatedAt      datetime not null on update CURRENT_TIMESTAMP,
    Expires        datetime not null
);

create table sessions
(
    token  char(43)     not null
        primary key,
    data   blob         not null,
    expiry timestamp(6) not null
);

create index sessions_expiry_idx
    on sessions (expiry);

create table users
(
    id              int auto_increment
        primary key,
    name            varchar(255)         not null,
    email           varchar(255)         not null,
    hashed_password char(60)             not null,
    created         datetime             not null,
    admin           tinyint(1) default 0 not null,
    user            tinyint(1)           not null,
    guest           tinyint(1)           not null,
    disabled        tinyint(1)           not null,
    constraint users_uc_email
        unique (email)
);

