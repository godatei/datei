-- Generated via: mise run import-db-schema
--
-- PostgreSQL database dump
--


-- Dumped from database version 18.3
-- Dumped by pg_dump version 18.3

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
-- Name: datei_permission_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.datei_permission_type AS ENUM (
    'owner',
    'read_write',
    'read_only'
);


--
-- Name: public_link_permission_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.public_link_permission_type AS ENUM (
    'read_only',
    'read_write'
);


--
-- Name: user_group_role; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.user_group_role AS ENUM (
    'admin',
    'member'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: datei_annotation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.datei_annotation (
    id uuid DEFAULT uuidv7() NOT NULL,
    datei_id uuid NOT NULL,
    key text NOT NULL,
    value text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: datei_comment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.datei_comment (
    id uuid DEFAULT uuidv7() NOT NULL,
    datei_id uuid NOT NULL,
    user_account_id uuid NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: datei_event; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.datei_event (
    id bigint NOT NULL,
    stream_id uuid NOT NULL,
    stream_version integer NOT NULL,
    event_type character varying NOT NULL,
    event_data jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_event_stream_version CHECK ((stream_version > 0))
);


--
-- Name: datei_event_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.datei_event_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: datei_event_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.datei_event_id_seq OWNED BY public.datei_event.id;


--
-- Name: datei_label; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.datei_label (
    datei_id uuid NOT NULL,
    label_id uuid NOT NULL
);


--
-- Name: datei_permission; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.datei_permission (
    id uuid DEFAULT uuidv7() NOT NULL,
    datei_id uuid NOT NULL,
    user_account_id uuid,
    user_group_id uuid,
    permission_type public.datei_permission_type NOT NULL,
    is_favorite boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_datei_permission_grantee CHECK ((((user_account_id IS NOT NULL) AND (user_group_id IS NULL)) OR ((user_account_id IS NULL) AND (user_group_id IS NOT NULL))))
);


--
-- Name: datei_permission_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.datei_permission_projection (
    id uuid NOT NULL,
    datei_id uuid NOT NULL,
    user_account_id uuid,
    user_group_id uuid,
    permission_type public.datei_permission_type NOT NULL,
    is_favorite boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT ck_datei_permission_projection_grantee CHECK ((((user_account_id IS NOT NULL) AND (user_group_id IS NULL)) OR ((user_account_id IS NULL) AND (user_group_id IS NOT NULL))))
);


--
-- Name: datei_projection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.datei_projection (
    id uuid NOT NULL,
    parent_id uuid,
    is_directory boolean DEFAULT false NOT NULL,
    linked_datei_id uuid,
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
-- Name: label; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.label (
    id uuid DEFAULT uuidv7() NOT NULL,
    name text NOT NULL,
    foreground_color text DEFAULT '#FFFFFF'::text NOT NULL,
    background_color text DEFAULT '#000000'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: public_link; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.public_link (
    id uuid DEFAULT uuidv7() NOT NULL,
    token text NOT NULL,
    created_by uuid NOT NULL,
    permission_type public.public_link_permission_type DEFAULT 'read_only'::public.public_link_permission_type NOT NULL,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: public_link_datei; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.public_link_datei (
    public_link_id uuid NOT NULL,
    datei_id uuid NOT NULL
);


--
-- Name: user_account; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_account (
    id uuid DEFAULT uuidv7() NOT NULL,
    name text NOT NULL,
    password_hash bytea NOT NULL,
    password_salt bytea NOT NULL,
    mfa_secret text,
    mfa_enabled boolean DEFAULT false NOT NULL,
    mfa_enabled_at timestamp with time zone,
    archived_at timestamp with time zone,
    last_logged_in_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_user_account_mfa CHECK (((mfa_enabled = false) OR (mfa_secret IS NOT NULL)))
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
-- Name: user_account_mfa_recovery_code; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_account_mfa_recovery_code (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_account_id uuid NOT NULL,
    code_hash bytea NOT NULL,
    code_salt bytea NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_email; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_email (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_account_id uuid NOT NULL,
    email text NOT NULL,
    verified_at timestamp with time zone,
    is_primary boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_group (
    id uuid DEFAULT uuidv7() NOT NULL,
    name text NOT NULL,
    created_by uuid NOT NULL,
    archived_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_group_member; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_group_member (
    user_account_id uuid NOT NULL,
    user_group_id uuid NOT NULL,
    role public.user_group_role DEFAULT 'member'::public.user_group_role NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: datei_event id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.datei_event ALTER COLUMN id SET DEFAULT nextval('public.datei_event_id_seq'::regclass);


--
-- Name: user_account_event id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_account_event ALTER COLUMN id SET DEFAULT nextval('public.user_account_event_id_seq'::regclass);


--
-- PostgreSQL database dump complete
--


