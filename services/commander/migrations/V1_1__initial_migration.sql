create table if not exists bot_users
(
    twitch_user_id varchar(36) not null,
    twitch_login   varchar(64) not null,
    primary key (twitch_user_id)
);
create table if not exists bot_commands
(
    twitch_user_id           varchar(36) not null,
    command_name             varchar(64) not null,
    command_cooldown_seconds int         not null,
    message_format           text        not null,
    twitch_response_type     text        not null,
    rl_platform              text        not null,
    rl_username              text        not null,
    primary key (twitch_user_id, command_name)
);