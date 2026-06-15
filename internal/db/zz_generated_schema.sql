-- Generated via: mise run import-db-schema
--
-- PostgreSQL database dump
--


-- Dumped from database version 18.4
-- Dumped by pg_dump version 18.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS '';


--
-- Name: file_permission_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.file_permission_type AS ENUM (
    'owner',
    'read_write',
    'read_only'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: file_event; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file_event (
    id bigint NOT NULL,
    stream_id uuid NOT NULL,
    stream_version integer NOT NULL,
    event_type character varying NOT NULL,
    event_data jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_event_stream_version CHECK ((stream_version > 0))
);


--
-- Name: file_event_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.file_event_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: file_event_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.file_event_id_seq OWNED BY public.file_event.id;


--
-- Name: file_permission_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file_permission_projection (
    id uuid NOT NULL,
    file_id uuid NOT NULL,
    user_account_id uuid,
    user_group_id uuid,
    permission_type public.file_permission_type NOT NULL,
    is_favorite boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT ck_file_permission_projection_grantee CHECK ((((user_account_id IS NOT NULL) AND (user_group_id IS NULL)) OR ((user_account_id IS NULL) AND (user_group_id IS NOT NULL))))
);


--
-- Name: file_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file_projection (
    id uuid NOT NULL,
    parent_id uuid,
    is_directory boolean DEFAULT false NOT NULL,
    linked_file_id uuid,
    name text NOT NULL,
    s3_key text,
    size bigint,
    checksum text,
    mime_type text,
    content_md text,
    content_search tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, COALESCE(content_md, ''::text))) STORED,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    trashed_at timestamp with time zone,
    created_by uuid,
    updated_by uuid,
    trashed_by uuid
);


--
-- Name: link_event; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.link_event (
    id bigint NOT NULL,
    stream_id uuid NOT NULL,
    stream_version integer NOT NULL,
    event_type character varying NOT NULL,
    event_data jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_link_event_stream_version CHECK ((stream_version > 0))
);


--
-- Name: link_event_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.link_event_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: link_event_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.link_event_id_seq OWNED BY public.link_event.id;


--
-- Name: link_file_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.link_file_projection (
    link_id uuid NOT NULL,
    file_id uuid NOT NULL,
    added_at timestamp with time zone NOT NULL
);


--
-- Name: link_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.link_projection (
    id uuid NOT NULL,
    owner_id uuid NOT NULL,
    name text NOT NULL,
    key text NOT NULL,
    code text,
    expires_at timestamp with time zone,
    revoked_at timestamp with time zone,
    open_count bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT ck_link_projection_code_length CHECK (((code IS NULL) OR ((length(code) >= 1) AND (length(code) <= 128)))),
    CONSTRAINT ck_link_projection_name_length CHECK (((length(name) >= 1) AND (length(name) <= 255)))
);


--
-- Name: user_account_email_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_account_email_projection (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_account_id uuid NOT NULL,
    email text NOT NULL,
    verified_at timestamp with time zone,
    is_primary boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_account_event; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_account_event (
    id bigint NOT NULL,
    stream_id uuid NOT NULL,
    stream_version integer NOT NULL,
    event_type character varying NOT NULL,
    event_data jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_user_account_event_stream_version CHECK ((stream_version > 0))
);


--
-- Name: user_account_event_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_account_event_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_account_event_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_account_event_id_seq OWNED BY public.user_account_event.id;


--
-- Name: user_account_mfa_recovery_code_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_account_mfa_recovery_code_projection (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_account_id uuid CONSTRAINT user_account_mfa_recovery_code_project_user_account_id_not_null NOT NULL,
    code_hash bytea NOT NULL,
    code_salt bytea NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_account_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_account_projection (
    id uuid DEFAULT uuidv7() NOT NULL,
    name text NOT NULL,
    password_hash bytea NOT NULL,
    password_salt bytea NOT NULL,
    is_admin boolean DEFAULT true NOT NULL,
    mfa_secret text,
    mfa_enabled boolean DEFAULT false NOT NULL,
    mfa_enabled_at timestamp with time zone,
    archived_at timestamp with time zone,
    last_logged_in_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_user_account_projection_mfa CHECK (((mfa_enabled = false) OR (mfa_secret IS NOT NULL)))
);


--
-- Name: file_event id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_event ALTER COLUMN id SET DEFAULT nextval('public.file_event_id_seq'::regclass);


--
-- Name: link_event id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.link_event ALTER COLUMN id SET DEFAULT nextval('public.link_event_id_seq'::regclass);


--
-- Name: user_account_event id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_account_event ALTER COLUMN id SET DEFAULT nextval('public.user_account_event_id_seq'::regclass);


--
-- PostgreSQL database dump complete
--


