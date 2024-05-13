create table public.users
(
    userid            integer      not null
        constraint person_pk
            primary key,
    balance integer          not null
);

alter table public.person
    owner to postgres;