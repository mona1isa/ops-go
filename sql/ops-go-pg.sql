-- Ops-Go PostgreSQL 初始化 DDL（由 cmd/pgdump 导出，共 40 张表）
-- 生成时间: 2026-10-23 10:39:27
-- 执行: psql -v ON_ERROR_STOP=1 -U <user> -d <dbname> -f ops-go-pg.sql
-- 说明: 应用启动时 GORM AutoMigrate 也会建表，本文件供全新环境一键初始化

SET client_min_messages = warning;

DROP TABLE IF EXISTS "casbin_rule" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "casbin_rule_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "casbin_rule" (
    "id" bigint NOT NULL,
    "ptype" character varying(100),
    "v0" character varying(100),
    "v1" character varying(100),
    "v2" character varying(100),
    "v3" character varying(100),
    "v4" character varying(100),
    "v5" character varying(100)
);
ALTER SEQUENCE "casbin_rule_id_seq" OWNED BY "casbin_rule"."id";
ALTER TABLE "casbin_rule" ALTER COLUMN "id" SET DEFAULT nextval('casbin_rule_id_seq'::regclass);
CREATE UNIQUE INDEX idx_casbin_rule ON public.casbin_rule USING btree (ptype, v0, v1, v2, v3, v4, v5);

DROP TABLE IF EXISTS "ops_cmdb_asset" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_cmdb_asset_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_cmdb_asset" (
    "model_type" character varying(32) NOT NULL,
    "name" character varying(128) NOT NULL,
    "sn" character varying(128),
    "ip" character varying(64),
    "status" smallint DEFAULT 1,
    "owner" character varying(64),
    "provider" character varying(128),
    "location" character varying(128),
    "expired_at" timestamp with time zone,
    "attrs" text,
    "source" character varying(32) DEFAULT 'manual'::character varying,
    "instance_id" integer DEFAULT 0,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_cmdb_asset_id_seq" OWNED BY "ops_cmdb_asset"."id";
ALTER TABLE "ops_cmdb_asset" ALTER COLUMN "id" SET DEFAULT nextval('ops_cmdb_asset_id_seq'::regclass);
COMMENT ON COLUMN "ops_cmdb_asset"."model_type" IS '资产模型类型';
COMMENT ON COLUMN "ops_cmdb_asset"."name" IS '资产名称';
COMMENT ON COLUMN "ops_cmdb_asset"."sn" IS '资产编号';
COMMENT ON COLUMN "ops_cmdb_asset"."ip" IS 'IP地址（域名/证书可为空）';
COMMENT ON COLUMN "ops_cmdb_asset"."status" IS '生命周期（1采购 2上架 3运行 4维护 5退役）';
COMMENT ON COLUMN "ops_cmdb_asset"."owner" IS '负责人';
COMMENT ON COLUMN "ops_cmdb_asset"."provider" IS '供应商/云厂商';
COMMENT ON COLUMN "ops_cmdb_asset"."location" IS '机房/区域';
COMMENT ON COLUMN "ops_cmdb_asset"."expired_at" IS '到期时间（域名/证书）';
COMMENT ON COLUMN "ops_cmdb_asset"."attrs" IS '模型专属属性（JSON字符串）';
COMMENT ON COLUMN "ops_cmdb_asset"."source" IS '来源（manual 手动 import 主机导入）';
COMMENT ON COLUMN "ops_cmdb_asset"."instance_id" IS '来源主机ID（导入资产，0 表示无）';
COMMENT ON COLUMN "ops_cmdb_asset"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_cmdb_asset"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_cmdb_asset"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_cmdb_asset"."remark" IS '备注';
CREATE INDEX idx_cmdb_asset_ip ON public.ops_cmdb_asset USING btree (ip);
CREATE UNIQUE INDEX uk_cmdb_asset_model_name ON public.ops_cmdb_asset USING btree (model_type, name);

DROP TABLE IF EXISTS "ops_cmdb_asset_tag" CASCADE;
CREATE TABLE "ops_cmdb_asset_tag" (
    "asset_id" integer NOT NULL,
    "tag_id" integer NOT NULL
);
COMMENT ON COLUMN "ops_cmdb_asset_tag"."asset_id" IS '资产ID';
COMMENT ON COLUMN "ops_cmdb_asset_tag"."tag_id" IS '标签ID';
CREATE UNIQUE INDEX uk_cmdb_asset_tag ON public.ops_cmdb_asset_tag USING btree (asset_id, tag_id);

DROP TABLE IF EXISTS "ops_cmdb_change_log" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_cmdb_change_log_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_cmdb_change_log" (
    "asset_id" integer NOT NULL,
    "action" character varying(32) NOT NULL,
    "detail" character varying(1000),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_cmdb_change_log_id_seq" OWNED BY "ops_cmdb_change_log"."id";
ALTER TABLE "ops_cmdb_change_log" ALTER COLUMN "id" SET DEFAULT nextval('ops_cmdb_change_log_id_seq'::regclass);
COMMENT ON COLUMN "ops_cmdb_change_log"."asset_id" IS '资产ID';
COMMENT ON COLUMN "ops_cmdb_change_log"."action" IS '动作（create/update/status/import/relation/tag/delete）';
COMMENT ON COLUMN "ops_cmdb_change_log"."detail" IS '变更摘要';
COMMENT ON COLUMN "ops_cmdb_change_log"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_cmdb_change_log"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_cmdb_change_log"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_cmdb_change_log"."remark" IS '备注';
CREATE INDEX idx_cmdb_changelog_asset ON public.ops_cmdb_change_log USING btree (asset_id);

DROP TABLE IF EXISTS "ops_cmdb_relation" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_cmdb_relation_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_cmdb_relation" (
    "source_id" integer NOT NULL,
    "target_id" integer NOT NULL,
    "relation_type" character varying(32) NOT NULL,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_cmdb_relation_id_seq" OWNED BY "ops_cmdb_relation"."id";
ALTER TABLE "ops_cmdb_relation" ALTER COLUMN "id" SET DEFAULT nextval('ops_cmdb_relation_id_seq'::regclass);
COMMENT ON COLUMN "ops_cmdb_relation"."source_id" IS '源资产ID';
COMMENT ON COLUMN "ops_cmdb_relation"."target_id" IS '目标资产ID';
COMMENT ON COLUMN "ops_cmdb_relation"."relation_type" IS '关系类型';
COMMENT ON COLUMN "ops_cmdb_relation"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_cmdb_relation"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_cmdb_relation"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_cmdb_relation"."remark" IS '备注';
CREATE INDEX idx_cmdb_relation_source ON public.ops_cmdb_relation USING btree (source_id);
CREATE INDEX idx_cmdb_relation_target ON public.ops_cmdb_relation USING btree (target_id);
CREATE UNIQUE INDEX uk_cmdb_relation ON public.ops_cmdb_relation USING btree (source_id, target_id, relation_type);

DROP TABLE IF EXISTS "ops_cmdb_subnet" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_cmdb_subnet_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_cmdb_subnet" (
    "name" character varying(64) NOT NULL,
    "cidr" character varying(64) NOT NULL,
    "gateway" character varying(64),
    "vlan" character varying(32),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_cmdb_subnet_id_seq" OWNED BY "ops_cmdb_subnet"."id";
ALTER TABLE "ops_cmdb_subnet" ALTER COLUMN "id" SET DEFAULT nextval('ops_cmdb_subnet_id_seq'::regclass);
COMMENT ON COLUMN "ops_cmdb_subnet"."name" IS '网段名称';
COMMENT ON COLUMN "ops_cmdb_subnet"."cidr" IS 'CIDR，如 10.0.1.0/24';
COMMENT ON COLUMN "ops_cmdb_subnet"."gateway" IS '网关';
COMMENT ON COLUMN "ops_cmdb_subnet"."vlan" IS 'VLAN';
COMMENT ON COLUMN "ops_cmdb_subnet"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_cmdb_subnet"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_cmdb_subnet"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_cmdb_subnet"."remark" IS '备注';
CREATE UNIQUE INDEX uk_cmdb_subnet_name ON public.ops_cmdb_subnet USING btree (name);

DROP TABLE IF EXISTS "ops_cmdb_tag" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_cmdb_tag_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_cmdb_tag" (
    "name" character varying(64) NOT NULL,
    "color" character varying(32),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_cmdb_tag_id_seq" OWNED BY "ops_cmdb_tag"."id";
ALTER TABLE "ops_cmdb_tag" ALTER COLUMN "id" SET DEFAULT nextval('ops_cmdb_tag_id_seq'::regclass);
COMMENT ON COLUMN "ops_cmdb_tag"."name" IS '标签名称';
COMMENT ON COLUMN "ops_cmdb_tag"."color" IS '展示颜色';
COMMENT ON COLUMN "ops_cmdb_tag"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_cmdb_tag"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_cmdb_tag"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_cmdb_tag"."remark" IS '备注';
CREATE UNIQUE INDEX uk_cmdb_tag_name ON public.ops_cmdb_tag USING btree (name);

DROP TABLE IF EXISTS "ops_command_block_log" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_command_block_log_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_command_block_log" (
    "id" bigint NOT NULL,
    "session_id" character varying(64),
    "channel" smallint DEFAULT 1,
    "user_id" bigint,
    "user_name" character varying(32),
    "instance_id" bigint,
    "instance_name" character varying(32),
    "instance_ip" character varying(32),
    "key_user" character varying(32),
    "command" character varying(1000),
    "normalized" character varying(1000),
    "rule_id" bigint,
    "rule_name" character varying(64),
    "created_at" timestamp with time zone NOT NULL
);
ALTER SEQUENCE "ops_command_block_log_id_seq" OWNED BY "ops_command_block_log"."id";
ALTER TABLE "ops_command_block_log" ALTER COLUMN "id" SET DEFAULT nextval('ops_command_block_log_id_seq'::regclass);
COMMENT ON COLUMN "ops_command_block_log"."session_id" IS '会话ID';
COMMENT ON COLUMN "ops_command_block_log"."channel" IS '接入渠道 1 SSH堡垒机 2 Web终端';
COMMENT ON COLUMN "ops_command_block_log"."user_id" IS '平台用户ID';
COMMENT ON COLUMN "ops_command_block_log"."user_name" IS '平台登录名';
COMMENT ON COLUMN "ops_command_block_log"."instance_id" IS '主机ID';
COMMENT ON COLUMN "ops_command_block_log"."instance_name" IS '主机名称';
COMMENT ON COLUMN "ops_command_block_log"."instance_ip" IS '主机IP';
COMMENT ON COLUMN "ops_command_block_log"."key_user" IS '远端登录账号';
COMMENT ON COLUMN "ops_command_block_log"."command" IS '被拦截的命令原文';
COMMENT ON COLUMN "ops_command_block_log"."normalized" IS '归一化后命中规则的形态';
COMMENT ON COLUMN "ops_command_block_log"."rule_id" IS '命中的规则ID';
COMMENT ON COLUMN "ops_command_block_log"."rule_name" IS '命中的规则名称';
COMMENT ON COLUMN "ops_command_block_log"."created_at" IS '拦截时间';
CREATE INDEX idx_ops_command_block_log_created_at ON public.ops_command_block_log USING btree (created_at);
CREATE INDEX idx_ops_command_block_log_instance_id ON public.ops_command_block_log USING btree (instance_id);
CREATE INDEX idx_ops_command_block_log_rule_id ON public.ops_command_block_log USING btree (rule_id);
CREATE INDEX idx_ops_command_block_log_session_id ON public.ops_command_block_log USING btree (session_id);
CREATE INDEX idx_ops_command_block_log_user_id ON public.ops_command_block_log USING btree (user_id);

DROP TABLE IF EXISTS "ops_dangerous_command" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_dangerous_command_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_dangerous_command" (
    "name" character varying(64) NOT NULL,
    "pattern" character varying(255) NOT NULL,
    "match_type" smallint DEFAULT 1,
    "description" character varying(255),
    "is_enabled" smallint DEFAULT 1,
    "is_builtin" smallint DEFAULT 0,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_dangerous_command_id_seq" OWNED BY "ops_dangerous_command"."id";
ALTER TABLE "ops_dangerous_command" ALTER COLUMN "id" SET DEFAULT nextval('ops_dangerous_command_id_seq'::regclass);
COMMENT ON COLUMN "ops_dangerous_command"."name" IS '规则名称';
COMMENT ON COLUMN "ops_dangerous_command"."pattern" IS '匹配模式';
COMMENT ON COLUMN "ops_dangerous_command"."match_type" IS '匹配类型 1精确匹配 2前缀匹配 3正则匹配';
COMMENT ON COLUMN "ops_dangerous_command"."description" IS '描述说明';
COMMENT ON COLUMN "ops_dangerous_command"."is_enabled" IS '是否启用 0禁用 1启用';
COMMENT ON COLUMN "ops_dangerous_command"."is_builtin" IS '是否内置 0自定义 1内置';
COMMENT ON COLUMN "ops_dangerous_command"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_dangerous_command"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_dangerous_command"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_dangerous_command"."remark" IS '备注';

DROP TABLE IF EXISTS "ops_execution_hosts" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_execution_hosts_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_execution_hosts" (
    "id" bigint NOT NULL,
    "execution_id" bigint NOT NULL,
    "step_exec_id" integer DEFAULT 0,
    "instance_id" bigint NOT NULL,
    "instance_name" character varying(32) NOT NULL,
    "instance_ip" character varying(32) NOT NULL,
    "key_id" bigint NOT NULL,
    "key_name" character varying(32) NOT NULL,
    "key_user" character varying(32) NOT NULL,
    "status" smallint DEFAULT 1,
    "result" text,
    "error_msg" text,
    "started_at" timestamp with time zone,
    "finished_at" timestamp with time zone,
    "duration" integer DEFAULT 0,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone
);
ALTER SEQUENCE "ops_execution_hosts_id_seq" OWNED BY "ops_execution_hosts"."id";
ALTER TABLE "ops_execution_hosts" ALTER COLUMN "id" SET DEFAULT nextval('ops_execution_hosts_id_seq'::regclass);
COMMENT ON COLUMN "ops_execution_hosts"."execution_id" IS '执行记录ID';
COMMENT ON COLUMN "ops_execution_hosts"."step_exec_id" IS '编排步骤执行ID 0=非编排';
COMMENT ON COLUMN "ops_execution_hosts"."instance_id" IS '主机ID';
COMMENT ON COLUMN "ops_execution_hosts"."instance_name" IS '主机名称';
COMMENT ON COLUMN "ops_execution_hosts"."instance_ip" IS '主机IP';
COMMENT ON COLUMN "ops_execution_hosts"."key_id" IS '凭证ID';
COMMENT ON COLUMN "ops_execution_hosts"."key_name" IS '凭证名称';
COMMENT ON COLUMN "ops_execution_hosts"."key_user" IS '登录用户';
COMMENT ON COLUMN "ops_execution_hosts"."status" IS '状态 1待执行 2执行中 3成功 4失败 5超时 6跳过';
COMMENT ON COLUMN "ops_execution_hosts"."result" IS '执行结果输出';
COMMENT ON COLUMN "ops_execution_hosts"."error_msg" IS '错误信息';
COMMENT ON COLUMN "ops_execution_hosts"."started_at" IS '开始时间';
COMMENT ON COLUMN "ops_execution_hosts"."finished_at" IS '结束时间';
COMMENT ON COLUMN "ops_execution_hosts"."duration" IS '执行耗时(毫秒)';
CREATE INDEX idx_ops_execution_hosts_execution_id ON public.ops_execution_hosts USING btree (execution_id);
CREATE INDEX idx_ops_execution_hosts_instance_id ON public.ops_execution_hosts USING btree (instance_id);

DROP TABLE IF EXISTS "ops_group" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_group_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_group" (
    "name" character varying(32) NOT NULL,
    "parent_id" integer DEFAULT 0,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_group_id_seq" OWNED BY "ops_group"."id";
ALTER TABLE "ops_group" ALTER COLUMN "id" SET DEFAULT nextval('ops_group_id_seq'::regclass);
COMMENT ON COLUMN "ops_group"."name" IS '名称';
COMMENT ON COLUMN "ops_group"."parent_id" IS '父级ID';
COMMENT ON COLUMN "ops_group"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_group"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_group"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_group"."remark" IS '备注';

DROP TABLE IF EXISTS "ops_instance" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_instance_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_instance" (
    "instance_id" character varying(32) NOT NULL,
    "name" character varying(32) NOT NULL,
    "cpu" integer NOT NULL,
    "mem_mb" integer NOT NULL,
    "disk_gb" integer NOT NULL,
    "ip" character varying(32) NOT NULL,
    "os" character varying(32) NOT NULL,
    "status" character varying(1) DEFAULT '1'::character varying,
    "online_status" character varying(1) DEFAULT '0'::character varying,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_instance_id_seq" OWNED BY "ops_instance"."id";
ALTER TABLE "ops_instance" ALTER COLUMN "id" SET DEFAULT nextval('ops_instance_id_seq'::regclass);
COMMENT ON COLUMN "ops_instance"."instance_id" IS '实例ID';
COMMENT ON COLUMN "ops_instance"."name" IS '主机名称';
COMMENT ON COLUMN "ops_instance"."cpu" IS 'CPU核数';
COMMENT ON COLUMN "ops_instance"."mem_mb" IS '内存大小(MB)';
COMMENT ON COLUMN "ops_instance"."disk_gb" IS '磁盘大小(GB)';
COMMENT ON COLUMN "ops_instance"."ip" IS 'IP地址';
COMMENT ON COLUMN "ops_instance"."os" IS '操作系统';
COMMENT ON COLUMN "ops_instance"."status" IS '状态（1 正常 0 禁用）';
COMMENT ON COLUMN "ops_instance"."online_status" IS '在线状态（1 在线 0 离线）';
COMMENT ON COLUMN "ops_instance"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_instance"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_instance"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_instance"."remark" IS '备注';

DROP TABLE IF EXISTS "ops_instance_group" CASCADE;
CREATE TABLE "ops_instance_group" (
    "instance_id" bigint NOT NULL,
    "group_id" bigint NOT NULL
);
COMMENT ON COLUMN "ops_instance_group"."instance_id" IS '实例ID';
COMMENT ON COLUMN "ops_instance_group"."group_id" IS '分组ID';

DROP TABLE IF EXISTS "ops_instance_keys" CASCADE;
CREATE TABLE "ops_instance_keys" (
    "instance_id" integer NOT NULL,
    "key_id" integer NOT NULL
);
COMMENT ON COLUMN "ops_instance_keys"."instance_id" IS '主机ID';
COMMENT ON COLUMN "ops_instance_keys"."key_id" IS '密钥ID';
CREATE INDEX idx_instance_id_key_id ON public.ops_instance_keys USING btree (instance_id, key_id);

DROP TABLE IF EXISTS "ops_key" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_key_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_key" (
    "name" character varying(128) NOT NULL,
    "user" character varying(128) NOT NULL,
    "credentials" text NOT NULL,
    "status" character varying(1) DEFAULT '1'::character varying,
    "protocol" character varying(16) NOT NULL,
    "port" integer NOT NULL,
    "type" integer NOT NULL,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_key_id_seq" OWNED BY "ops_key"."id";
ALTER TABLE "ops_key" ALTER COLUMN "id" SET DEFAULT nextval('ops_key_id_seq'::regclass);
COMMENT ON COLUMN "ops_key"."name" IS '名称';
COMMENT ON COLUMN "ops_key"."user" IS '登录用户';
COMMENT ON COLUMN "ops_key"."credentials" IS '登录凭证';
COMMENT ON COLUMN "ops_key"."status" IS '状态（1 正常 0 禁用）';
COMMENT ON COLUMN "ops_key"."protocol" IS '协议: SSH or RDP';
COMMENT ON COLUMN "ops_key"."port" IS '端口';
COMMENT ON COLUMN "ops_key"."type" IS '类型: 1 密码 2 秘钥';
COMMENT ON COLUMN "ops_key"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_key"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_key"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_key"."remark" IS '备注';
CREATE UNIQUE INDEX uk_name ON public.ops_key USING btree (name);

DROP TABLE IF EXISTS "ops_log_alert_rule" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_log_alert_rule_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_log_alert_rule" (
    "name" character varying(128) NOT NULL,
    "selector" character varying(512),
    "host" character varying(64),
    "source" character varying(64),
    "keyword" character varying(255),
    "level" character varying(16),
    "window_sec" integer DEFAULT 300,
    "threshold" integer DEFAULT 10,
    "severity" smallint DEFAULT 2,
    "cooldown_sec" integer DEFAULT 300,
    "silence_until" bigint DEFAULT 0,
    "muted" character varying(1) DEFAULT '0'::character varying,
    "channel_ids" character varying(255),
    "group_by_host" character varying(1) DEFAULT '1'::character varying,
    "enabled" character varying(1) DEFAULT '1'::character varying,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_log_alert_rule_id_seq" OWNED BY "ops_log_alert_rule"."id";
ALTER TABLE "ops_log_alert_rule" ALTER COLUMN "id" SET DEFAULT nextval('ops_log_alert_rule_id_seq'::regclass);
COMMENT ON COLUMN "ops_log_alert_rule"."name" IS '规则名称';
COMMENT ON COLUMN "ops_log_alert_rule"."selector" IS '流选择器，如 {job="ops_log",host="10.0.0.1"}，为空按下方条件自动生成';
COMMENT ON COLUMN "ops_log_alert_rule"."host" IS '主机过滤（空为全部主机）';
COMMENT ON COLUMN "ops_log_alert_rule"."source" IS '业务来源标签过滤';
COMMENT ON COLUMN "ops_log_alert_rule"."keyword" IS '关键字过滤（生成 |= 过滤器）';
COMMENT ON COLUMN "ops_log_alert_rule"."level" IS '级别过滤，如 ERROR/WARN，为空不过滤';
COMMENT ON COLUMN "ops_log_alert_rule"."window_sec" IS '统计时间窗(秒)';
COMMENT ON COLUMN "ops_log_alert_rule"."threshold" IS '窗口内命中条数阈值';
COMMENT ON COLUMN "ops_log_alert_rule"."severity" IS '级别（1 紧急 2 警告 3 提醒）';
COMMENT ON COLUMN "ops_log_alert_rule"."cooldown_sec" IS '重复告警冷却(秒)';
COMMENT ON COLUMN "ops_log_alert_rule"."silence_until" IS '静默截止时间(Unix秒)，0 表示不静默';
COMMENT ON COLUMN "ops_log_alert_rule"."muted" IS '静音（1 只记录事件不通知 0 正常）';
COMMENT ON COLUMN "ops_log_alert_rule"."channel_ids" IS '通知渠道ID列表(逗号分隔)';
COMMENT ON COLUMN "ops_log_alert_rule"."group_by_host" IS '按主机分别告警（1 是 0 汇总）';
COMMENT ON COLUMN "ops_log_alert_rule"."enabled" IS '启用状态（1 启用 0 停用）';
COMMENT ON COLUMN "ops_log_alert_rule"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_log_alert_rule"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_log_alert_rule"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_log_alert_rule"."remark" IS '备注';

DROP TABLE IF EXISTS "ops_log_source" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_log_source_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_log_source" (
    "name" character varying(128) NOT NULL,
    "instance_id" integer NOT NULL,
    "instance_name" character varying(32) NOT NULL,
    "instance_ip" character varying(32) NOT NULL,
    "key_id" integer DEFAULT 0,
    "path" character varying(512) NOT NULL,
    "job" character varying(64) DEFAULT 'ops_log'::character varying,
    "source" character varying(64),
    "labels" character varying(512),
    "level_regex" character varying(255),
    "enabled" character varying(1) DEFAULT '1'::character varying,
    "interval_sec" integer DEFAULT 15,
    "timeout_sec" integer DEFAULT 20,
    "max_bytes" integer DEFAULT 0,
    "back_bytes" integer DEFAULT 0,
    "last_inode" bigint DEFAULT 0,
    "last_pos" bigint DEFAULT 0,
    "last_file" character varying(512),
    "last_lines" integer DEFAULT 0,
    "last_drop" bigint DEFAULT 0,
    "last_collect_at" timestamp with time zone,
    "last_status" smallint DEFAULT 0,
    "last_error" character varying(500),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_log_source_id_seq" OWNED BY "ops_log_source"."id";
ALTER TABLE "ops_log_source" ALTER COLUMN "id" SET DEFAULT nextval('ops_log_source_id_seq'::regclass);
COMMENT ON COLUMN "ops_log_source"."name" IS '采集任务名称';
COMMENT ON COLUMN "ops_log_source"."instance_id" IS '主机ID';
COMMENT ON COLUMN "ops_log_source"."instance_name" IS '主机名称';
COMMENT ON COLUMN "ops_log_source"."instance_ip" IS '主机IP';
COMMENT ON COLUMN "ops_log_source"."key_id" IS '采集凭证ID（0 自动选择）';
COMMENT ON COLUMN "ops_log_source"."path" IS '日志文件路径，支持 * ? 通配符';
COMMENT ON COLUMN "ops_log_source"."job" IS 'Loki job 标签';
COMMENT ON COLUMN "ops_log_source"."source" IS '业务来源标签，如 nginx/access';
COMMENT ON COLUMN "ops_log_source"."labels" IS '附加标签(JSON 对象)';
COMMENT ON COLUMN "ops_log_source"."level_regex" IS '级别提取正则（需包含捕获组），为空时用内置 ERROR/WARN/INFO 识别';
COMMENT ON COLUMN "ops_log_source"."enabled" IS '采集开关（1 启用 0 停用）';
COMMENT ON COLUMN "ops_log_source"."interval_sec" IS '采集间隔(秒)';
COMMENT ON COLUMN "ops_log_source"."timeout_sec" IS '单次拉取超时(秒)';
COMMENT ON COLUMN "ops_log_source"."max_bytes" IS '单轮最大拉取字节(0 用全局默认)';
COMMENT ON COLUMN "ops_log_source"."back_bytes" IS '首次接入回溯字节(0 用全局默认)';
COMMENT ON COLUMN "ops_log_source"."last_inode" IS '上次读取的文件 inode，用于识别轮转';
COMMENT ON COLUMN "ops_log_source"."last_pos" IS '下次读取的起始字节偏移';
COMMENT ON COLUMN "ops_log_source"."last_file" IS '上次实际读取的文件';
COMMENT ON COLUMN "ops_log_source"."last_lines" IS '上次采集行数';
COMMENT ON COLUMN "ops_log_source"."last_drop" IS '上次因限速丢弃的字节数';
COMMENT ON COLUMN "ops_log_source"."last_collect_at" IS '最近采集成功时间';
COMMENT ON COLUMN "ops_log_source"."last_status" IS '采集状态（0 未采集 1 成功 2 失败）';
COMMENT ON COLUMN "ops_log_source"."last_error" IS '最近采集失败原因';
COMMENT ON COLUMN "ops_log_source"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_log_source"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_log_source"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_log_source"."remark" IS '备注';
CREATE INDEX idx_log_source_instance ON public.ops_log_source USING btree (instance_id);

DROP TABLE IF EXISTS "ops_monitor_alert_event" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_monitor_alert_event_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_monitor_alert_event" (
    "id" bigint NOT NULL,
    "rule_id" bigint NOT NULL,
    "rule_name" character varying(128),
    "fingerprint" character varying(64) NOT NULL,
    "instance_ip" character varying(32),
    "labels" character varying(512),
    "value" numeric(18,4),
    "threshold" numeric(18,4),
    "severity" smallint DEFAULT 2,
    "status" smallint DEFAULT 1,
    "pending_since" timestamp with time zone,
    "fired_at" timestamp with time zone,
    "resolved_at" timestamp with time zone,
    "last_notify_at" timestamp with time zone,
    "notify_count" integer DEFAULT 0,
    "suppressed" integer DEFAULT 0,
    "summary" character varying(512),
    "source" character varying(16) DEFAULT 'monitor'::character varying,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone
);
ALTER SEQUENCE "ops_monitor_alert_event_id_seq" OWNED BY "ops_monitor_alert_event"."id";
ALTER TABLE "ops_monitor_alert_event" ALTER COLUMN "id" SET DEFAULT nextval('ops_monitor_alert_event_id_seq'::regclass);
COMMENT ON COLUMN "ops_monitor_alert_event"."rule_id" IS '规则ID';
COMMENT ON COLUMN "ops_monitor_alert_event"."rule_name" IS '规则名称（快照）';
COMMENT ON COLUMN "ops_monitor_alert_event"."fingerprint" IS '序列指纹（规则ID+标签集）';
COMMENT ON COLUMN "ops_monitor_alert_event"."instance_ip" IS '触发主机IP';
COMMENT ON COLUMN "ops_monitor_alert_event"."labels" IS '触发时的标签集';
COMMENT ON COLUMN "ops_monitor_alert_event"."value" IS '触发值';
COMMENT ON COLUMN "ops_monitor_alert_event"."threshold" IS '阈值（快照）';
COMMENT ON COLUMN "ops_monitor_alert_event"."severity" IS '级别（1 紧急 2 警告 3 提醒）';
COMMENT ON COLUMN "ops_monitor_alert_event"."status" IS '状态（1 待确认 2 告警中 3 已恢复）';
COMMENT ON COLUMN "ops_monitor_alert_event"."pending_since" IS '进入待确认时间';
COMMENT ON COLUMN "ops_monitor_alert_event"."fired_at" IS '触发时间';
COMMENT ON COLUMN "ops_monitor_alert_event"."resolved_at" IS '恢复时间';
COMMENT ON COLUMN "ops_monitor_alert_event"."last_notify_at" IS '最近通知时间（冷却判定依据）';
COMMENT ON COLUMN "ops_monitor_alert_event"."notify_count" IS '累计通知次数';
COMMENT ON COLUMN "ops_monitor_alert_event"."suppressed" IS '累计被冷却/静默抑制的次数';
COMMENT ON COLUMN "ops_monitor_alert_event"."summary" IS '告警摘要';
COMMENT ON COLUMN "ops_monitor_alert_event"."source" IS '告警来源（monitor 指标 log 日志）';
CREATE INDEX idx_ops_monitor_alert_event_fingerprint ON public.ops_monitor_alert_event USING btree (fingerprint);
CREATE INDEX idx_ops_monitor_alert_event_last_notify_at ON public.ops_monitor_alert_event USING btree (last_notify_at);
CREATE INDEX idx_ops_monitor_alert_event_rule_id ON public.ops_monitor_alert_event USING btree (rule_id);
CREATE INDEX idx_ops_monitor_alert_event_source ON public.ops_monitor_alert_event USING btree (source);
CREATE INDEX idx_ops_monitor_alert_event_status ON public.ops_monitor_alert_event USING btree (status);

DROP TABLE IF EXISTS "ops_monitor_notify_channel" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_monitor_notify_channel_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_monitor_notify_channel" (
    "name" character varying(128) NOT NULL,
    "type" smallint DEFAULT 1,
    "url" character varying(512) NOT NULL,
    "secret" character varying(255),
    "headers" text,
    "enabled" character varying(1) DEFAULT '1'::character varying,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_monitor_notify_channel_id_seq" OWNED BY "ops_monitor_notify_channel"."id";
ALTER TABLE "ops_monitor_notify_channel" ALTER COLUMN "id" SET DEFAULT nextval('ops_monitor_notify_channel_id_seq'::regclass);
COMMENT ON COLUMN "ops_monitor_notify_channel"."name" IS '渠道名称';
COMMENT ON COLUMN "ops_monitor_notify_channel"."type" IS '类型（1 Webhook 2 邮件 3 钉钉 4 企业微信 5 飞书）';
COMMENT ON COLUMN "ops_monitor_notify_channel"."url" IS '回调地址（邮件渠道存收件人，逗号分隔）';
COMMENT ON COLUMN "ops_monitor_notify_channel"."secret" IS 'HMAC-SHA256 签名密钥';
COMMENT ON COLUMN "ops_monitor_notify_channel"."headers" IS '自定义请求头（JSON 对象）';
COMMENT ON COLUMN "ops_monitor_notify_channel"."enabled" IS '启用状态（1 启用 0 停用）';
COMMENT ON COLUMN "ops_monitor_notify_channel"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_monitor_notify_channel"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_monitor_notify_channel"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_monitor_notify_channel"."remark" IS '备注';

DROP TABLE IF EXISTS "ops_monitor_notify_log" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_monitor_notify_log_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_monitor_notify_log" (
    "id" bigint NOT NULL,
    "event_id" bigint,
    "rule_id" bigint,
    "channel_id" bigint,
    "channel_name" character varying(128),
    "recovered" smallint DEFAULT 0,
    "payload" text,
    "status_code" integer,
    "response_body" character varying(1024),
    "attempt" integer DEFAULT 1,
    "success" smallint DEFAULT 0,
    "error_msg" character varying(512),
    "created_at" timestamp with time zone
);
ALTER SEQUENCE "ops_monitor_notify_log_id_seq" OWNED BY "ops_monitor_notify_log"."id";
ALTER TABLE "ops_monitor_notify_log" ALTER COLUMN "id" SET DEFAULT nextval('ops_monitor_notify_log_id_seq'::regclass);
COMMENT ON COLUMN "ops_monitor_notify_log"."event_id" IS '告警事件ID';
COMMENT ON COLUMN "ops_monitor_notify_log"."rule_id" IS '规则ID';
COMMENT ON COLUMN "ops_monitor_notify_log"."channel_id" IS '渠道ID';
COMMENT ON COLUMN "ops_monitor_notify_log"."channel_name" IS '渠道名称（快照）';
COMMENT ON COLUMN "ops_monitor_notify_log"."recovered" IS '事件类型（0 告警触发 1 告警恢复）';
COMMENT ON COLUMN "ops_monitor_notify_log"."payload" IS '请求体';
COMMENT ON COLUMN "ops_monitor_notify_log"."status_code" IS 'HTTP 响应码';
COMMENT ON COLUMN "ops_monitor_notify_log"."response_body" IS '响应体（截断）';
COMMENT ON COLUMN "ops_monitor_notify_log"."attempt" IS '第几次尝试';
COMMENT ON COLUMN "ops_monitor_notify_log"."success" IS '是否成功（1 成功 0 失败）';
COMMENT ON COLUMN "ops_monitor_notify_log"."error_msg" IS '失败原因';
CREATE INDEX idx_ops_monitor_notify_log_channel_id ON public.ops_monitor_notify_log USING btree (channel_id);
CREATE INDEX idx_ops_monitor_notify_log_event_id ON public.ops_monitor_notify_log USING btree (event_id);
CREATE INDEX idx_ops_monitor_notify_log_rule_id ON public.ops_monitor_notify_log USING btree (rule_id);

DROP TABLE IF EXISTS "ops_monitor_rule" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_monitor_rule_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_monitor_rule" (
    "name" character varying(128) NOT NULL,
    "metric" character varying(64),
    "prom_ql" character varying(512) NOT NULL,
    "operator" character varying(4) DEFAULT '>'::character varying,
    "threshold" numeric(18,4) DEFAULT 0,
    "severity" smallint DEFAULT 2,
    "for_sec" integer DEFAULT 0,
    "cooldown_sec" integer DEFAULT 300,
    "silence_until" bigint DEFAULT 0,
    "muted" character varying(1) DEFAULT '0'::character varying,
    "channel_ids" character varying(255),
    "enabled" character varying(1) DEFAULT '1'::character varying,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_monitor_rule_id_seq" OWNED BY "ops_monitor_rule"."id";
ALTER TABLE "ops_monitor_rule" ALTER COLUMN "id" SET DEFAULT nextval('ops_monitor_rule_id_seq'::regclass);
COMMENT ON COLUMN "ops_monitor_rule"."name" IS '规则名称';
COMMENT ON COLUMN "ops_monitor_rule"."metric" IS '指标名称（展示用）';
COMMENT ON COLUMN "ops_monitor_rule"."prom_ql" IS 'PromQL 表达式，可用 {{instance}} 占位按主机展开';
COMMENT ON COLUMN "ops_monitor_rule"."operator" IS '比较符（> >= < <= == !=），PromQL 未写比较时生效';
COMMENT ON COLUMN "ops_monitor_rule"."threshold" IS '阈值';
COMMENT ON COLUMN "ops_monitor_rule"."severity" IS '级别（1 紧急 2 警告 3 提醒）';
COMMENT ON COLUMN "ops_monitor_rule"."for_sec" IS '持续时长(秒)后才触发';
COMMENT ON COLUMN "ops_monitor_rule"."cooldown_sec" IS '重复告警冷却(秒)';
COMMENT ON COLUMN "ops_monitor_rule"."silence_until" IS '静默截止时间(Unix秒)，0 表示不静默';
COMMENT ON COLUMN "ops_monitor_rule"."muted" IS '静音（1 只记录事件不通知 0 正常）';
COMMENT ON COLUMN "ops_monitor_rule"."channel_ids" IS '通知渠道ID列表(逗号分隔)';
COMMENT ON COLUMN "ops_monitor_rule"."enabled" IS '启用状态（1 启用 0 停用）';
COMMENT ON COLUMN "ops_monitor_rule"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_monitor_rule"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_monitor_rule"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_monitor_rule"."remark" IS '备注';

DROP TABLE IF EXISTS "ops_monitor_target" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_monitor_target_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_monitor_target" (
    "instance_id" integer NOT NULL,
    "instance_name" character varying(32) NOT NULL,
    "instance_ip" character varying(32) NOT NULL,
    "key_id" integer DEFAULT 0,
    "enabled" character varying(1) DEFAULT '1'::character varying,
    "interval_sec" integer DEFAULT 60,
    "timeout_sec" integer DEFAULT 15,
    "last_collect_at" timestamp with time zone,
    "last_status" smallint DEFAULT 0,
    "last_error" character varying(500),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_monitor_target_id_seq" OWNED BY "ops_monitor_target"."id";
ALTER TABLE "ops_monitor_target" ALTER COLUMN "id" SET DEFAULT nextval('ops_monitor_target_id_seq'::regclass);
COMMENT ON COLUMN "ops_monitor_target"."instance_id" IS '主机ID';
COMMENT ON COLUMN "ops_monitor_target"."instance_name" IS '主机名称';
COMMENT ON COLUMN "ops_monitor_target"."instance_ip" IS 'IP地址';
COMMENT ON COLUMN "ops_monitor_target"."key_id" IS '采集凭证ID（0 自动选择）';
COMMENT ON COLUMN "ops_monitor_target"."enabled" IS '采集开关（1 启用 0 停用）';
COMMENT ON COLUMN "ops_monitor_target"."interval_sec" IS '采集间隔(秒)';
COMMENT ON COLUMN "ops_monitor_target"."timeout_sec" IS '采集超时(秒)';
COMMENT ON COLUMN "ops_monitor_target"."last_collect_at" IS '最近采集成功时间';
COMMENT ON COLUMN "ops_monitor_target"."last_status" IS '采集状态（0 未采集 1 成功 2 失败）';
COMMENT ON COLUMN "ops_monitor_target"."last_error" IS '最近采集失败原因';
COMMENT ON COLUMN "ops_monitor_target"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_monitor_target"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_monitor_target"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_monitor_target"."remark" IS '备注';
CREATE INDEX idx_monitor_target_instance ON public.ops_monitor_target USING btree (instance_id);

DROP TABLE IF EXISTS "ops_pipeline_steps" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_pipeline_steps_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_pipeline_steps" (
    "pipeline_id" integer NOT NULL,
    "step_name" character varying(128) NOT NULL,
    "template_id" integer NOT NULL,
    "step_order" integer DEFAULT 0,
    "parent_step_id" integer DEFAULT 0,
    "condition" character varying(255),
    "on_failure" smallint DEFAULT 1,
    "retry_count" integer DEFAULT 0,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_pipeline_steps_id_seq" OWNED BY "ops_pipeline_steps"."id";
ALTER TABLE "ops_pipeline_steps" ALTER COLUMN "id" SET DEFAULT nextval('ops_pipeline_steps_id_seq'::regclass);
COMMENT ON COLUMN "ops_pipeline_steps"."pipeline_id" IS '编排ID';
COMMENT ON COLUMN "ops_pipeline_steps"."step_name" IS '步骤名称';
COMMENT ON COLUMN "ops_pipeline_steps"."template_id" IS '任务模板ID';
COMMENT ON COLUMN "ops_pipeline_steps"."step_order" IS '执行顺序';
COMMENT ON COLUMN "ops_pipeline_steps"."parent_step_id" IS '父步骤ID 0=顶层';
COMMENT ON COLUMN "ops_pipeline_steps"."condition" IS '执行条件';
COMMENT ON COLUMN "ops_pipeline_steps"."on_failure" IS '失败策略 1终止 2跳过 3重试';
COMMENT ON COLUMN "ops_pipeline_steps"."retry_count" IS '重试次数';
COMMENT ON COLUMN "ops_pipeline_steps"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_pipeline_steps"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_pipeline_steps"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_pipeline_steps"."remark" IS '备注';
CREATE INDEX idx_ops_pipeline_steps_pipeline_id ON public.ops_pipeline_steps USING btree (pipeline_id);

DROP TABLE IF EXISTS "ops_script" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_script_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_script" (
    "name" character varying(128) NOT NULL,
    "content" text,
    "type" character varying(20) DEFAULT 'bash'::character varying,
    "remark" character varying(500),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying
);
ALTER SEQUENCE "ops_script_id_seq" OWNED BY "ops_script"."id";
ALTER TABLE "ops_script" ALTER COLUMN "id" SET DEFAULT nextval('ops_script_id_seq'::regclass);
COMMENT ON COLUMN "ops_script"."name" IS '脚本名称';
COMMENT ON COLUMN "ops_script"."content" IS '脚本内容';
COMMENT ON COLUMN "ops_script"."type" IS '脚本类型';
COMMENT ON COLUMN "ops_script"."remark" IS '备注';
COMMENT ON COLUMN "ops_script"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_script"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_script"."del_flag" IS '删除标识（0正常 1 已删除）';

DROP TABLE IF EXISTS "ops_session_record" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_session_record_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_session_record" (
    "id" bigint NOT NULL,
    "session_id" character varying(64) NOT NULL,
    "user_id" bigint NOT NULL,
    "instance_id" bigint NOT NULL,
    "instance_name" character varying(32) NOT NULL,
    "instance_ip" character varying(32) NOT NULL,
    "key_id" bigint NOT NULL,
    "key_name" character varying(32) NOT NULL,
    "key_user" character varying(32) NOT NULL,
    "start_time" timestamp with time zone NOT NULL,
    "end_time" timestamp with time zone,
    "duration" bigint DEFAULT 0,
    "status" smallint DEFAULT 1,
    "recording_file" character varying(255),
    "file_size" bigint DEFAULT 0,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone
);
ALTER SEQUENCE "ops_session_record_id_seq" OWNED BY "ops_session_record"."id";
ALTER TABLE "ops_session_record" ALTER COLUMN "id" SET DEFAULT nextval('ops_session_record_id_seq'::regclass);
COMMENT ON COLUMN "ops_session_record"."session_id" IS '会话ID';
COMMENT ON COLUMN "ops_session_record"."user_id" IS '用户ID';
COMMENT ON COLUMN "ops_session_record"."instance_id" IS '主机ID';
COMMENT ON COLUMN "ops_session_record"."instance_name" IS '主机名称';
COMMENT ON COLUMN "ops_session_record"."instance_ip" IS '主机IP';
COMMENT ON COLUMN "ops_session_record"."key_id" IS '凭证ID';
COMMENT ON COLUMN "ops_session_record"."key_name" IS '凭证名称';
COMMENT ON COLUMN "ops_session_record"."key_user" IS '登录用户';
COMMENT ON COLUMN "ops_session_record"."start_time" IS '开始时间';
COMMENT ON COLUMN "ops_session_record"."end_time" IS '结束时间';
COMMENT ON COLUMN "ops_session_record"."duration" IS '会话时长（秒）';
COMMENT ON COLUMN "ops_session_record"."status" IS '状态（1 进行中 2 已结束 3 异常中断）';
COMMENT ON COLUMN "ops_session_record"."recording_file" IS '录像文件路径';
COMMENT ON COLUMN "ops_session_record"."file_size" IS '文件大小（字节）';
CREATE INDEX idx_ops_session_record_instance_id ON public.ops_session_record USING btree (instance_id);
CREATE UNIQUE INDEX idx_ops_session_record_session_id ON public.ops_session_record USING btree (session_id);
CREATE INDEX idx_ops_session_record_start_time ON public.ops_session_record USING btree (start_time);
CREATE INDEX idx_ops_session_record_user_id ON public.ops_session_record USING btree (user_id);

DROP TABLE IF EXISTS "ops_step_executions" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_step_executions_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_step_executions" (
    "id" bigint NOT NULL,
    "execution_id" bigint NOT NULL,
    "step_id" bigint NOT NULL,
    "step_name" character varying(128) NOT NULL,
    "template_id" bigint NOT NULL,
    "status" smallint DEFAULT 1,
    "started_at" timestamp with time zone,
    "finished_at" timestamp with time zone,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone
);
ALTER SEQUENCE "ops_step_executions_id_seq" OWNED BY "ops_step_executions"."id";
ALTER TABLE "ops_step_executions" ALTER COLUMN "id" SET DEFAULT nextval('ops_step_executions_id_seq'::regclass);
COMMENT ON COLUMN "ops_step_executions"."execution_id" IS '执行记录ID';
COMMENT ON COLUMN "ops_step_executions"."step_id" IS '编排步骤ID';
COMMENT ON COLUMN "ops_step_executions"."step_name" IS '步骤名称';
COMMENT ON COLUMN "ops_step_executions"."template_id" IS '任务模板ID';
COMMENT ON COLUMN "ops_step_executions"."status" IS '状态 1待执行 2执行中 3成功 4失败 5跳过';
COMMENT ON COLUMN "ops_step_executions"."started_at" IS '开始时间';
COMMENT ON COLUMN "ops_step_executions"."finished_at" IS '结束时间';
CREATE INDEX idx_ops_step_executions_execution_id ON public.ops_step_executions USING btree (execution_id);

DROP TABLE IF EXISTS "ops_task_executions" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_task_executions_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_task_executions" (
    "id" bigint NOT NULL,
    "execution_no" character varying(64) NOT NULL,
    "name" character varying(128) NOT NULL,
    "type" smallint NOT NULL,
    "source_id" integer,
    "user_id" bigint NOT NULL,
    "user_name" character varying(32) NOT NULL,
    "status" smallint DEFAULT 1,
    "total_hosts" integer DEFAULT 0,
    "success_hosts" integer DEFAULT 0,
    "fail_hosts" integer DEFAULT 0,
    "timeout" integer DEFAULT 300,
    "content" text,
    "script_lang" character varying(32),
    "src_path" character varying(512),
    "dest_path" character varying(512),
    "started_at" timestamp with time zone,
    "finished_at" timestamp with time zone,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone
);
ALTER SEQUENCE "ops_task_executions_id_seq" OWNED BY "ops_task_executions"."id";
ALTER TABLE "ops_task_executions" ALTER COLUMN "id" SET DEFAULT nextval('ops_task_executions_id_seq'::regclass);
COMMENT ON COLUMN "ops_task_executions"."execution_no" IS '执行编号';
COMMENT ON COLUMN "ops_task_executions"."name" IS '执行名称';
COMMENT ON COLUMN "ops_task_executions"."type" IS '类型 1快速命令 2快速脚本 3快速文件 4模板执行 5编排执行';
COMMENT ON COLUMN "ops_task_executions"."source_id" IS '来源ID 模板ID/编排ID';
COMMENT ON COLUMN "ops_task_executions"."user_id" IS '执行人ID';
COMMENT ON COLUMN "ops_task_executions"."user_name" IS '执行人用户名';
COMMENT ON COLUMN "ops_task_executions"."status" IS '状态 1待执行 2执行中 3已完成 4部分失败 5全部失败 6已取消';
COMMENT ON COLUMN "ops_task_executions"."total_hosts" IS '目标主机数';
COMMENT ON COLUMN "ops_task_executions"."success_hosts" IS '成功主机数';
COMMENT ON COLUMN "ops_task_executions"."fail_hosts" IS '失败主机数';
COMMENT ON COLUMN "ops_task_executions"."timeout" IS '超时时间(秒)';
COMMENT ON COLUMN "ops_task_executions"."content" IS '执行内容(快速执行时保存)';
COMMENT ON COLUMN "ops_task_executions"."script_lang" IS '脚本语言';
COMMENT ON COLUMN "ops_task_executions"."src_path" IS '源文件路径';
COMMENT ON COLUMN "ops_task_executions"."dest_path" IS '目标文件路径';
COMMENT ON COLUMN "ops_task_executions"."started_at" IS '开始时间';
COMMENT ON COLUMN "ops_task_executions"."finished_at" IS '结束时间';
CREATE UNIQUE INDEX idx_ops_task_executions_execution_no ON public.ops_task_executions USING btree (execution_no);
CREATE INDEX idx_ops_task_executions_user_id ON public.ops_task_executions USING btree (user_id);

DROP TABLE IF EXISTS "ops_task_pipelines" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_task_pipelines_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_task_pipelines" (
    "name" character varying(128) NOT NULL,
    "description" character varying(255),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_task_pipelines_id_seq" OWNED BY "ops_task_pipelines"."id";
ALTER TABLE "ops_task_pipelines" ALTER COLUMN "id" SET DEFAULT nextval('ops_task_pipelines_id_seq'::regclass);
COMMENT ON COLUMN "ops_task_pipelines"."name" IS '编排名称';
COMMENT ON COLUMN "ops_task_pipelines"."description" IS '描述';
COMMENT ON COLUMN "ops_task_pipelines"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_task_pipelines"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_task_pipelines"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_task_pipelines"."remark" IS '备注';

DROP TABLE IF EXISTS "ops_task_templates" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "ops_task_templates_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "ops_task_templates" (
    "name" character varying(128) NOT NULL,
    "type" smallint NOT NULL,
    "content" text,
    "script_lang" character varying(32),
    "src_path" character varying(512),
    "dest_path" character varying(512),
    "timeout" integer DEFAULT 300,
    "key_id" integer DEFAULT 0,
    "description" character varying(255),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "ops_task_templates_id_seq" OWNED BY "ops_task_templates"."id";
ALTER TABLE "ops_task_templates" ALTER COLUMN "id" SET DEFAULT nextval('ops_task_templates_id_seq'::regclass);
COMMENT ON COLUMN "ops_task_templates"."name" IS '模板名称';
COMMENT ON COLUMN "ops_task_templates"."type" IS '任务类型 1执行命令 2执行脚本 3分发文件';
COMMENT ON COLUMN "ops_task_templates"."content" IS '命令内容/脚本内容';
COMMENT ON COLUMN "ops_task_templates"."script_lang" IS '脚本语言 shell/python';
COMMENT ON COLUMN "ops_task_templates"."src_path" IS '源文件路径';
COMMENT ON COLUMN "ops_task_templates"."dest_path" IS '目标路径';
COMMENT ON COLUMN "ops_task_templates"."timeout" IS '超时时间(秒)';
COMMENT ON COLUMN "ops_task_templates"."key_id" IS '凭证ID 0自动选择';
COMMENT ON COLUMN "ops_task_templates"."description" IS '描述';
COMMENT ON COLUMN "ops_task_templates"."create_by" IS '创建人ID';
COMMENT ON COLUMN "ops_task_templates"."update_by" IS '修改人ID';
COMMENT ON COLUMN "ops_task_templates"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "ops_task_templates"."remark" IS '备注';

DROP TABLE IF EXISTS "ops_user_instance_auth" CASCADE;
CREATE TABLE "ops_user_instance_auth" (
    "user_id" integer NOT NULL,
    "instance_id" integer,
    "group_id" integer,
    "auth_type" integer NOT NULL,
    "del_flag" character varying(1) DEFAULT '0'::character varying
);
COMMENT ON COLUMN "ops_user_instance_auth"."user_id" IS '用户ID';
COMMENT ON COLUMN "ops_user_instance_auth"."instance_id" IS '主机ID';
COMMENT ON COLUMN "ops_user_instance_auth"."group_id" IS '分组ID';
COMMENT ON COLUMN "ops_user_instance_auth"."auth_type" IS '类型: 1 主机 2 分组';
COMMENT ON COLUMN "ops_user_instance_auth"."del_flag" IS '删除标识（0正常 1 已删除）';

DROP TABLE IF EXISTS "ops_user_instance_key_auth" CASCADE;
CREATE TABLE "ops_user_instance_key_auth" (
    "user_id" integer NOT NULL,
    "instance_id" integer NOT NULL,
    "group_id" integer NOT NULL,
    "key_id" integer NOT NULL,
    "auth_type" integer NOT NULL,
    "del_flag" character varying(1) DEFAULT '0'::character varying
);
COMMENT ON COLUMN "ops_user_instance_key_auth"."user_id" IS '用户ID';
COMMENT ON COLUMN "ops_user_instance_key_auth"."instance_id" IS '主机ID';
COMMENT ON COLUMN "ops_user_instance_key_auth"."group_id" IS '主机分组ID';
COMMENT ON COLUMN "ops_user_instance_key_auth"."key_id" IS '密钥ID';
COMMENT ON COLUMN "ops_user_instance_key_auth"."auth_type" IS '授权类型（1 主机 2 主机组）';
COMMENT ON COLUMN "ops_user_instance_key_auth"."del_flag" IS '删除标识（0正常 1 已删除）';

DROP TABLE IF EXISTS "sys_dept" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "sys_dept_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "sys_dept" (
    "name" character varying(32) NOT NULL,
    "parent_id" bigint,
    "order_num" integer,
    "status" boolean DEFAULT false,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "sys_dept_id_seq" OWNED BY "sys_dept"."id";
ALTER TABLE "sys_dept" ALTER COLUMN "id" SET DEFAULT nextval('sys_dept_id_seq'::regclass);
COMMENT ON COLUMN "sys_dept"."name" IS '部门名称';
COMMENT ON COLUMN "sys_dept"."parent_id" IS '父级ID';
COMMENT ON COLUMN "sys_dept"."order_num" IS '排序';
COMMENT ON COLUMN "sys_dept"."status" IS '状态（true正常 false停用）';
COMMENT ON COLUMN "sys_dept"."create_by" IS '创建人ID';
COMMENT ON COLUMN "sys_dept"."update_by" IS '修改人ID';
COMMENT ON COLUMN "sys_dept"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "sys_dept"."remark" IS '备注';

DROP TABLE IF EXISTS "sys_log" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "sys_log_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "sys_log" (
    "dept_id" bigint,
    "create_user" character varying(16),
    "method" character varying(8),
    "request_uri" character varying(255),
    "params" text,
    "resp" text,
    "ip_addr" character varying(20),
    "status_code" character(8),
    "cost_time_ms" bigint,
    "oper_module" character varying(32),
    "oper_type" character varying(32),
    "sensitive" character varying(1) DEFAULT '0'::character varying,
    "biz_code" character varying(16),
    "biz_msg" character varying(255),
    "user_agent" character varying(512),
    "operate_at" character varying(20),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "sys_log_id_seq" OWNED BY "sys_log"."id";
ALTER TABLE "sys_log" ALTER COLUMN "id" SET DEFAULT nextval('sys_log_id_seq'::regclass);
COMMENT ON COLUMN "sys_log"."dept_id" IS '创建人部门';
COMMENT ON COLUMN "sys_log"."create_user" IS '创建人用户名';
COMMENT ON COLUMN "sys_log"."method" IS '请求方法';
COMMENT ON COLUMN "sys_log"."request_uri" IS '请求地址';
COMMENT ON COLUMN "sys_log"."params" IS '请求参数';
COMMENT ON COLUMN "sys_log"."resp" IS 'resp';
COMMENT ON COLUMN "sys_log"."ip_addr" IS '请求IP';
COMMENT ON COLUMN "sys_log"."status_code" IS '响应状态码';
COMMENT ON COLUMN "sys_log"."cost_time_ms" IS '响应耗时';
COMMENT ON COLUMN "sys_log"."oper_module" IS '操作模块';
COMMENT ON COLUMN "sys_log"."oper_type" IS '操作类型（query/add/edit/delete/grant/other）';
COMMENT ON COLUMN "sys_log"."sensitive" IS '敏感操作（1 是 0 否）';
COMMENT ON COLUMN "sys_log"."biz_code" IS '业务返回码';
COMMENT ON COLUMN "sys_log"."biz_msg" IS '业务返回信息';
COMMENT ON COLUMN "sys_log"."user_agent" IS '客户端UA';
COMMENT ON COLUMN "sys_log"."operate_at" IS '操作时间(格式化)';
COMMENT ON COLUMN "sys_log"."create_by" IS '创建人ID';
COMMENT ON COLUMN "sys_log"."update_by" IS '修改人ID';
COMMENT ON COLUMN "sys_log"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "sys_log"."remark" IS '备注';
CREATE INDEX idx_sys_log_oper_module ON public.sys_log USING btree (oper_module);
CREATE INDEX idx_sys_log_oper_type ON public.sys_log USING btree (oper_type);
CREATE INDEX idx_sys_log_sensitive ON public.sys_log USING btree (sensitive);

DROP TABLE IF EXISTS "sys_login_log" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "sys_login_log_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "sys_login_log" (
    "id" bigint NOT NULL,
    "user_name" character varying(64),
    "user_id" integer DEFAULT 0,
    "result" smallint DEFAULT 0,
    "reason" character varying(255),
    "ip_addr" character varying(64),
    "location" character varying(128),
    "user_agent" character varying(512),
    "browser" character varying(64),
    "os" character varying(64),
    "platform" character varying(32),
    "dept_id" integer DEFAULT 0,
    "created_at" timestamp with time zone
);
ALTER SEQUENCE "sys_login_log_id_seq" OWNED BY "sys_login_log"."id";
ALTER TABLE "sys_login_log" ALTER COLUMN "id" SET DEFAULT nextval('sys_login_log_id_seq'::regclass);
COMMENT ON COLUMN "sys_login_log"."user_name" IS '登录用户名';
COMMENT ON COLUMN "sys_login_log"."user_id" IS '用户ID（用户不存在时为0）';
COMMENT ON COLUMN "sys_login_log"."result" IS '结果（0 失败 1 成功 2 登出）';
COMMENT ON COLUMN "sys_login_log"."reason" IS '失败原因';
COMMENT ON COLUMN "sys_login_log"."ip_addr" IS '来源IP';
COMMENT ON COLUMN "sys_login_log"."location" IS '来源IP归属（预留）';
COMMENT ON COLUMN "sys_login_log"."user_agent" IS '客户端UA';
COMMENT ON COLUMN "sys_login_log"."browser" IS '浏览器';
COMMENT ON COLUMN "sys_login_log"."os" IS '操作系统';
COMMENT ON COLUMN "sys_login_log"."platform" IS '登录端（web/bastion）';
COMMENT ON COLUMN "sys_login_log"."dept_id" IS '部门ID';
COMMENT ON COLUMN "sys_login_log"."created_at" IS '登录时间';
CREATE INDEX idx_sys_login_log_created_at ON public.sys_login_log USING btree (created_at);
CREATE INDEX idx_sys_login_log_ip_addr ON public.sys_login_log USING btree (ip_addr);
CREATE INDEX idx_sys_login_log_result ON public.sys_login_log USING btree (result);
CREATE INDEX idx_sys_login_log_user_name ON public.sys_login_log USING btree (user_name);

DROP TABLE IF EXISTS "sys_menu" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "sys_menu_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "sys_menu" (
    "name" character varying(64) NOT NULL,
    "parent_id" integer,
    "order_num" integer,
    "path" character varying(200),
    "component" character varying(255),
    "is_affix" boolean,
    "is_iframe" boolean,
    "is_link" boolean,
    "keep_alive" boolean,
    "type" character(1),
    "is_hide" boolean,
    "status" boolean DEFAULT false,
    "url" character varying(128) DEFAULT NULL::character varying,
    "perms" character varying(100),
    "icon" character varying(100),
    "request_url" character varying(100),
    "request_method" character varying(10),
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "sys_menu_id_seq" OWNED BY "sys_menu"."id";
ALTER TABLE "sys_menu" ALTER COLUMN "id" SET DEFAULT nextval('sys_menu_id_seq'::regclass);
COMMENT ON COLUMN "sys_menu"."name" IS '菜单名称';
COMMENT ON COLUMN "sys_menu"."parent_id" IS '父菜单ID';
COMMENT ON COLUMN "sys_menu"."order_num" IS '显示顺序';
COMMENT ON COLUMN "sys_menu"."path" IS '路由地址';
COMMENT ON COLUMN "sys_menu"."component" IS '组件路径';
COMMENT ON COLUMN "sys_menu"."is_affix" IS '是否固定（1 是 0 否）';
COMMENT ON COLUMN "sys_menu"."is_iframe" IS '是否为内链（1是 0否）';
COMMENT ON COLUMN "sys_menu"."is_link" IS '是否为外链（1是 0否）';
COMMENT ON COLUMN "sys_menu"."keep_alive" IS '是否缓存（true缓存 false不缓存）';
COMMENT ON COLUMN "sys_menu"."type" IS '菜单类型（M目录 C菜单 F按钮）';
COMMENT ON COLUMN "sys_menu"."is_hide" IS '是否隐藏（1隐藏 0显示）';
COMMENT ON COLUMN "sys_menu"."status" IS '菜单状态（1正常 0停用）';
COMMENT ON COLUMN "sys_menu"."url" IS '外链地址';
COMMENT ON COLUMN "sys_menu"."perms" IS '权限标识';
COMMENT ON COLUMN "sys_menu"."icon" IS '菜单图标';
COMMENT ON COLUMN "sys_menu"."request_url" IS '接口请求地址';
COMMENT ON COLUMN "sys_menu"."request_method" IS '接口请求方法';
COMMENT ON COLUMN "sys_menu"."create_by" IS '创建人ID';
COMMENT ON COLUMN "sys_menu"."update_by" IS '修改人ID';
COMMENT ON COLUMN "sys_menu"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "sys_menu"."remark" IS '备注';

DROP TABLE IF EXISTS "sys_role" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "sys_role_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "sys_role" (
    "name" character varying(255) NOT NULL,
    "order_num" integer DEFAULT 1,
    "status" character varying(1) DEFAULT '1'::character varying,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "sys_role_id_seq" OWNED BY "sys_role"."id";
ALTER TABLE "sys_role" ALTER COLUMN "id" SET DEFAULT nextval('sys_role_id_seq'::regclass);
COMMENT ON COLUMN "sys_role"."name" IS '角色名称';
COMMENT ON COLUMN "sys_role"."order_num" IS '排序';
COMMENT ON COLUMN "sys_role"."status" IS '状态（0：禁用，1：正常）';
COMMENT ON COLUMN "sys_role"."create_by" IS '创建人ID';
COMMENT ON COLUMN "sys_role"."update_by" IS '修改人ID';
COMMENT ON COLUMN "sys_role"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "sys_role"."remark" IS '备注';

DROP TABLE IF EXISTS "sys_role_menu" CASCADE;
CREATE TABLE "sys_role_menu" (
    "role_id" integer NOT NULL,
    "menu_id" integer NOT NULL
);

DROP TABLE IF EXISTS "sys_user" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "sys_user_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "sys_user" (
    "dept_id" bigint NOT NULL,
    "user_name" character varying(32) NOT NULL,
    "nick_name" character varying(32),
    "email" character varying(64),
    "phone" character varying(16),
    "sex" smallint,
    "avatar" character varying(255),
    "password" character varying(512) NOT NULL,
    "status" character varying(1) DEFAULT '0'::character varying,
    "login_ip" character varying(128),
    "login_date" timestamp with time zone,
    "totp_secret" character varying(512),
    "totp_enabled" smallint DEFAULT 0,
    "recovery_codes" text,
    "id" bigint NOT NULL,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "create_by" character varying(32),
    "update_by" character varying(32),
    "del_flag" character varying(1) DEFAULT '0'::character varying,
    "remark" character varying(500)
);
ALTER SEQUENCE "sys_user_id_seq" OWNED BY "sys_user"."id";
ALTER TABLE "sys_user" ALTER COLUMN "id" SET DEFAULT nextval('sys_user_id_seq'::regclass);
COMMENT ON COLUMN "sys_user"."dept_id" IS '部门ID';
COMMENT ON COLUMN "sys_user"."user_name" IS '用户名';
COMMENT ON COLUMN "sys_user"."nick_name" IS '昵称';
COMMENT ON COLUMN "sys_user"."email" IS '邮箱';
COMMENT ON COLUMN "sys_user"."phone" IS '电话';
COMMENT ON COLUMN "sys_user"."sex" IS '性别（0男 1女 2未知）';
COMMENT ON COLUMN "sys_user"."avatar" IS '头像地址';
COMMENT ON COLUMN "sys_user"."password" IS '密码';
COMMENT ON COLUMN "sys_user"."status" IS '帐号状态（1正常 0停用）';
COMMENT ON COLUMN "sys_user"."login_ip" IS '登录IP';
COMMENT ON COLUMN "sys_user"."login_date" IS '登录时间';
COMMENT ON COLUMN "sys_user"."totp_secret" IS 'TOTP密钥（加密存储）';
COMMENT ON COLUMN "sys_user"."totp_enabled" IS 'MFA状态（0未绑定 1已启用）';
COMMENT ON COLUMN "sys_user"."recovery_codes" IS '一次性恢复码（bcrypt哈希JSON数组）';
COMMENT ON COLUMN "sys_user"."create_by" IS '创建人ID';
COMMENT ON COLUMN "sys_user"."update_by" IS '修改人ID';
COMMENT ON COLUMN "sys_user"."del_flag" IS '删除标识（0正常 1 已删除）';
COMMENT ON COLUMN "sys_user"."remark" IS '备注';

DROP TABLE IF EXISTS "sys_user_role" CASCADE;
CREATE TABLE "sys_user_role" (
    "user_id" bigint NOT NULL,
    "role_id" bigint NOT NULL
);
COMMENT ON COLUMN "sys_user_role"."user_id" IS '用户ID';
COMMENT ON COLUMN "sys_user_role"."role_id" IS '角色ID';

DROP TABLE IF EXISTS "sys_user_token" CASCADE;
CREATE SEQUENCE IF NOT EXISTS "sys_user_token_id_seq" AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1 CACHE 1 NO CYCLE;
CREATE TABLE "sys_user_token" (
    "id" bigint NOT NULL,
    "user_id" bigint,
    "token" text,
    "created_at" timestamp with time zone,
    "expire_at" timestamp with time zone
);
ALTER SEQUENCE "sys_user_token_id_seq" OWNED BY "sys_user_token"."id";
ALTER TABLE "sys_user_token" ALTER COLUMN "id" SET DEFAULT nextval('sys_user_token_id_seq'::regclass);

-- 主键 / 唯一约束
ALTER TABLE "casbin_rule" ADD CONSTRAINT "casbin_rule_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_cmdb_asset" ADD CONSTRAINT "ops_cmdb_asset_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_cmdb_change_log" ADD CONSTRAINT "ops_cmdb_change_log_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_cmdb_relation" ADD CONSTRAINT "ops_cmdb_relation_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_cmdb_subnet" ADD CONSTRAINT "ops_cmdb_subnet_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_cmdb_tag" ADD CONSTRAINT "ops_cmdb_tag_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_command_block_log" ADD CONSTRAINT "ops_command_block_log_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_dangerous_command" ADD CONSTRAINT "ops_dangerous_command_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_execution_hosts" ADD CONSTRAINT "ops_execution_hosts_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_group" ADD CONSTRAINT "ops_group_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_group" ADD CONSTRAINT "uni_ops_group_name" UNIQUE (name);
ALTER TABLE "ops_instance" ADD CONSTRAINT "ops_instance_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_instance" ADD CONSTRAINT "uni_ops_instance_name" UNIQUE (name);
ALTER TABLE "ops_key" ADD CONSTRAINT "ops_key_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_log_alert_rule" ADD CONSTRAINT "ops_log_alert_rule_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_log_source" ADD CONSTRAINT "ops_log_source_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_monitor_alert_event" ADD CONSTRAINT "ops_monitor_alert_event_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_monitor_notify_channel" ADD CONSTRAINT "ops_monitor_notify_channel_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_monitor_notify_log" ADD CONSTRAINT "ops_monitor_notify_log_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_monitor_rule" ADD CONSTRAINT "ops_monitor_rule_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_monitor_target" ADD CONSTRAINT "ops_monitor_target_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_pipeline_steps" ADD CONSTRAINT "ops_pipeline_steps_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_script" ADD CONSTRAINT "ops_script_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_session_record" ADD CONSTRAINT "ops_session_record_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_step_executions" ADD CONSTRAINT "ops_step_executions_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_task_executions" ADD CONSTRAINT "ops_task_executions_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_task_pipelines" ADD CONSTRAINT "ops_task_pipelines_pkey" PRIMARY KEY (id);
ALTER TABLE "ops_task_templates" ADD CONSTRAINT "ops_task_templates_pkey" PRIMARY KEY (id);
ALTER TABLE "sys_dept" ADD CONSTRAINT "sys_dept_pkey" PRIMARY KEY (id);
ALTER TABLE "sys_dept" ADD CONSTRAINT "uni_sys_dept_name" UNIQUE (name);
ALTER TABLE "sys_log" ADD CONSTRAINT "sys_log_pkey" PRIMARY KEY (id);
ALTER TABLE "sys_login_log" ADD CONSTRAINT "sys_login_log_pkey" PRIMARY KEY (id);
ALTER TABLE "sys_menu" ADD CONSTRAINT "sys_menu_pkey" PRIMARY KEY (id);
ALTER TABLE "sys_menu" ADD CONSTRAINT "uni_sys_menu_name" UNIQUE (name);
ALTER TABLE "sys_role" ADD CONSTRAINT "sys_role_pkey" PRIMARY KEY (id);
ALTER TABLE "sys_user" ADD CONSTRAINT "sys_user_pkey" PRIMARY KEY (id);
ALTER TABLE "sys_user" ADD CONSTRAINT "uni_sys_user_user_name" UNIQUE (user_name);
ALTER TABLE "sys_user_token" ADD CONSTRAINT "sys_user_token_pkey" PRIMARY KEY (id);

-- 外键约束
ALTER TABLE "ops_pipeline_steps" ADD CONSTRAINT "fk_ops_task_pipelines_steps" FOREIGN KEY (pipeline_id) REFERENCES ops_task_pipelines(id);
