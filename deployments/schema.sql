-- SEQUENCE: public.users_id_seq

-- DROP SEQUENCE public.users_id_seq;

CREATE SEQUENCE public.users_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 9223372036854775807
    CACHE 1;

ALTER SEQUENCE public.users_id_seq
    OWNER TO kris;


-- Table: public.users

-- DROP TABLE public.users;

CREATE TABLE public.users
(
    id bigint NOT NULL DEFAULT nextval('users_id_seq'::regclass),
    username character(128) COLLATE pg_catalog."default" NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_username_key UNIQUE (username)
)

TABLESPACE pg_default;

ALTER TABLE public.users
    OWNER to kris;


-- SEQUENCE: public.chats_id_seq

-- DROP SEQUENCE public.chats_id_seq;

CREATE SEQUENCE public.chats_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 9223372036854775807
    CACHE 1;

ALTER SEQUENCE public.chats_id_seq
    OWNER TO kris;


-- Table: public.chats

-- DROP TABLE public.chats;

CREATE TABLE public.chats
(
    id bigint NOT NULL DEFAULT nextval('chats_id_seq'::regclass),
    name character(128) COLLATE pg_catalog."default" NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT chats_pkey PRIMARY KEY (id),
    CONSTRAINT chats_name_key UNIQUE (name)
)

    TABLESPACE pg_default;

ALTER TABLE public.chats
    OWNER to kris;


-- Table: public.chat_users

-- DROP TABLE public.chat_users;

CREATE TABLE public.chat_users
(
    chat_id bigint NOT NULL,
    user_id bigint NOT NULL,
    CONSTRAINT chat_users_pkey PRIMARY KEY (chat_id, user_id),
    CONSTRAINT "chat-users_chat_id_fkey" FOREIGN KEY (chat_id)
        REFERENCES public.chats (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID,
    CONSTRAINT "chat-users_user_id_fkey" FOREIGN KEY (user_id)
        REFERENCES public.users (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

    TABLESPACE pg_default;

ALTER TABLE public.chat_users
    OWNER to kris;


-- SEQUENCE: public.messages_id_seq

-- DROP SEQUENCE public.messages_id_seq;

CREATE SEQUENCE public.messages_id_seq
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 9223372036854775807
    CACHE 1;

ALTER SEQUENCE public.messages_id_seq
    OWNER TO kris;


-- Table: public.messages

-- DROP TABLE public.messages;

CREATE TABLE public.messages
(
    id bigint NOT NULL DEFAULT nextval('messages_id_seq'::regclass),
    chat_id bigint NOT NULL,
    author_id bigint NOT NULL,
    text text COLLATE pg_catalog."default" NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT messages_pkey PRIMARY KEY (id),
    CONSTRAINT messages_author_id_fkey FOREIGN KEY (author_id)
        REFERENCES public.users (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT messages_chat_id_author_id_fkey FOREIGN KEY (author_id, chat_id)
        REFERENCES public.chat_users (user_id, chat_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT messages_chat_id_fkey FOREIGN KEY (chat_id)
        REFERENCES public.chats (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

    TABLESPACE pg_default;

ALTER TABLE public.messages
    OWNER to kris;