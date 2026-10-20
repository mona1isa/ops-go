-- Ops-Go 数据库初始化 DDL（由 cmd/sqldump 从运行库导出，共 39 张表）
-- 生成时间: 2026-10-20 17:03:37
-- 执行: mysql -u<user> -p < ops-go.sql
-- 说明: 应用启动时 GORM AutoMigrate 也会建表，本文件供全新环境一键初始化

CREATE DATABASE IF NOT EXISTS `ops-go` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `ops-go`;

DROP TABLE IF EXISTS `casbin_rule`;
CREATE TABLE `casbin_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ptype` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `v0` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `v1` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `v2` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `v3` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `v4` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `v5` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
) ENGINE=InnoDB AUTO_INCREMENT=560 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_cmdb_asset`;
CREATE TABLE `ops_cmdb_asset` (
  `model_type` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '资产模型类型',
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '资产名称',
  `sn` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '资产编号',
  `ip` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'IP地址（域名/证书可为空）',
  `status` tinyint DEFAULT '1' COMMENT '生命周期（1采购 2上架 3运行 4维护 5退役）',
  `owner` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '负责人',
  `provider` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '供应商/云厂商',
  `location` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '机房/区域',
  `expired_at` datetime(3) DEFAULT NULL COMMENT '到期时间（域名/证书）',
  `attrs` text COLLATE utf8mb4_general_ci COMMENT '模型专属属性（JSON字符串）',
  `source` varchar(32) COLLATE utf8mb4_general_ci DEFAULT 'manual' COMMENT '来源（manual 手动 import 主机导入）',
  `instance_id` int DEFAULT '0' COMMENT '来源主机ID（导入资产，0 表示无）',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_asset_model_name` (`model_type`,`name`),
  KEY `idx_cmdb_asset_ip` (`ip`)
) ENGINE=InnoDB AUTO_INCREMENT=29 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_cmdb_asset_tag`;
CREATE TABLE `ops_cmdb_asset_tag` (
  `asset_id` int NOT NULL COMMENT '资产ID',
  `tag_id` int NOT NULL COMMENT '标签ID',
  UNIQUE KEY `uk_cmdb_asset_tag` (`asset_id`,`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_cmdb_change_log`;
CREATE TABLE `ops_cmdb_change_log` (
  `asset_id` int NOT NULL COMMENT '资产ID',
  `action` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '动作（create/update/status/import/relation/tag/delete）',
  `detail` varchar(1000) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '变更摘要',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  KEY `idx_cmdb_changelog_asset` (`asset_id`)
) ENGINE=InnoDB AUTO_INCREMENT=35 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_cmdb_relation`;
CREATE TABLE `ops_cmdb_relation` (
  `source_id` int NOT NULL COMMENT '源资产ID',
  `target_id` int NOT NULL COMMENT '目标资产ID',
  `relation_type` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '关系类型',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_relation` (`source_id`,`target_id`,`relation_type`),
  KEY `idx_cmdb_relation_source` (`source_id`),
  KEY `idx_cmdb_relation_target` (`target_id`)
) ENGINE=InnoDB AUTO_INCREMENT=29 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_cmdb_subnet`;
CREATE TABLE `ops_cmdb_subnet` (
  `name` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT '网段名称',
  `cidr` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'CIDR，如 10.0.1.0/24',
  `gateway` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '网关',
  `vlan` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'VLAN',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_subnet_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_cmdb_tag`;
CREATE TABLE `ops_cmdb_tag` (
  `name` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT '标签名称',
  `color` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '展示颜色',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_tag_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_dangerous_command`;
CREATE TABLE `ops_dangerous_command` (
  `name` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT '规则名称',
  `pattern` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '匹配模式',
  `match_type` tinyint DEFAULT '1' COMMENT '匹配类型 1精确匹配 2前缀匹配 3正则匹配',
  `description` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述说明',
  `is_enabled` tinyint DEFAULT '1' COMMENT '是否启用 0禁用 1启用',
  `is_builtin` tinyint DEFAULT '0' COMMENT '是否内置 0自定义 1内置',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_execution_hosts`;
CREATE TABLE `ops_execution_hosts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `execution_id` bigint unsigned NOT NULL COMMENT '执行记录ID',
  `step_exec_id` bigint DEFAULT '0' COMMENT '编排步骤执行ID 0=非编排',
  `instance_id` bigint NOT NULL COMMENT '主机ID',
  `instance_name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '主机名称',
  `instance_ip` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '主机IP',
  `key_id` bigint NOT NULL COMMENT '凭证ID',
  `key_name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '凭证名称',
  `key_user` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '登录用户',
  `status` tinyint DEFAULT '1' COMMENT '状态 1待执行 2执行中 3成功 4失败 5超时 6跳过',
  `result` text COLLATE utf8mb4_general_ci COMMENT '执行结果输出',
  `error_msg` text COLLATE utf8mb4_general_ci COMMENT '错误信息',
  `started_at` datetime(3) DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime(3) DEFAULT NULL COMMENT '结束时间',
  `duration` bigint DEFAULT '0' COMMENT '执行耗时(毫秒)',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ops_execution_hosts_execution_id` (`execution_id`),
  KEY `idx_ops_execution_hosts_instance_id` (`instance_id`)
) ENGINE=InnoDB AUTO_INCREMENT=38 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_group`;
CREATE TABLE `ops_group` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '名称',
  `parent_id` bigint DEFAULT '0' COMMENT '父级ID',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_ops_group_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_instance`;
CREATE TABLE `ops_instance` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `instance_id` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '实例ID',
  `name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '主机名称',
  `cpu` int NOT NULL COMMENT 'CPU核数',
  `mem_mb` int NOT NULL COMMENT '内存大小(MB)',
  `disk_gb` int NOT NULL COMMENT '磁盘大小(GB)',
  `ip` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'IP地址',
  `os` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '操作系统',
  `status` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '状态（1 正常 0 禁用）',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  `online_status` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '在线状态（1 在线 0 离线）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_ops_instance_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=68 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_instance_group`;
CREATE TABLE `ops_instance_group` (
  `instance_id` varchar(11) COLLATE utf8mb4_general_ci NOT NULL COMMENT '实例ID',
  `group_id` varchar(11) COLLATE utf8mb4_general_ci NOT NULL COMMENT '分组ID'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_instance_keys`;
CREATE TABLE `ops_instance_keys` (
  `instance_id` int NOT NULL COMMENT '主机ID',
  `key_id` int NOT NULL COMMENT '密钥ID',
  KEY `idx_instance_id_key_id` (`instance_id`,`key_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_key`;
CREATE TABLE `ops_key` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '名称',
  `user` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '登录用户',
  `credentials` text COLLATE utf8mb4_general_ci NOT NULL COMMENT '登录凭证',
  `status` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '状态（1 正常 0 禁用）',
  `protocol` varchar(16) COLLATE utf8mb4_general_ci NOT NULL COMMENT '协议: SSH or RDP',
  `port` int NOT NULL COMMENT '端口',
  `type` int NOT NULL COMMENT '类型: 1 密码 2 秘钥',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_log_alert_rule`;
CREATE TABLE `ops_log_alert_rule` (
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '规则名称',
  `selector` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '流选择器，如 {job="ops_log",host="10.0.0.1"}，为空按下方条件自动生成',
  `host` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '主机过滤（空为全部主机）',
  `source` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '业务来源标签过滤',
  `keyword` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '关键字过滤（生成 |= 过滤器）',
  `level` varchar(16) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '级别过滤，如 ERROR/WARN，为空不过滤',
  `window_sec` int DEFAULT '300' COMMENT '统计时间窗(秒)',
  `threshold` int DEFAULT '10' COMMENT '窗口内命中条数阈值',
  `severity` tinyint DEFAULT '2' COMMENT '级别（1 紧急 2 警告 3 提醒）',
  `cooldown_sec` int DEFAULT '300' COMMENT '重复告警冷却(秒)',
  `silence_until` bigint DEFAULT '0' COMMENT '静默截止时间(Unix秒)，0 表示不静默',
  `muted` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '静音（1 只记录事件不通知 0 正常）',
  `channel_ids` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '通知渠道ID列表(逗号分隔)',
  `group_by_host` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '按主机分别告警（1 是 0 汇总）',
  `enabled` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '启用状态（1 启用 0 停用）',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_log_source`;
CREATE TABLE `ops_log_source` (
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '采集任务名称',
  `instance_id` int NOT NULL COMMENT '主机ID',
  `instance_name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '主机名称',
  `instance_ip` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '主机IP',
  `key_id` int DEFAULT '0' COMMENT '采集凭证ID（0 自动选择）',
  `path` varchar(512) COLLATE utf8mb4_general_ci NOT NULL COMMENT '日志文件路径，支持 * ? 通配符',
  `job` varchar(64) COLLATE utf8mb4_general_ci DEFAULT 'ops_log' COMMENT 'Loki job 标签',
  `source` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '业务来源标签，如 nginx/access',
  `labels` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '附加标签(JSON 对象)',
  `level_regex` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '级别提取正则（需包含捕获组），为空时用内置 ERROR/WARN/INFO 识别',
  `enabled` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '采集开关（1 启用 0 停用）',
  `interval_sec` int DEFAULT '15' COMMENT '采集间隔(秒)',
  `timeout_sec` int DEFAULT '20' COMMENT '单次拉取超时(秒)',
  `max_bytes` int DEFAULT '0' COMMENT '单轮最大拉取字节(0 用全局默认)',
  `back_bytes` int DEFAULT '0' COMMENT '首次接入回溯字节(0 用全局默认)',
  `last_inode` bigint DEFAULT '0' COMMENT '上次读取的文件 inode，用于识别轮转',
  `last_pos` bigint DEFAULT '0' COMMENT '下次读取的起始字节偏移',
  `last_file` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '上次实际读取的文件',
  `last_lines` int DEFAULT '0' COMMENT '上次采集行数',
  `last_drop` bigint DEFAULT '0' COMMENT '上次因限速丢弃的字节数',
  `last_collect_at` datetime(3) DEFAULT NULL COMMENT '最近采集成功时间',
  `last_status` tinyint DEFAULT '0' COMMENT '采集状态（0 未采集 1 成功 2 失败）',
  `last_error` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '最近采集失败原因',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  KEY `idx_log_source_instance` (`instance_id`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_monitor_alert_event`;
CREATE TABLE `ops_monitor_alert_event` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `rule_id` bigint NOT NULL COMMENT '规则ID',
  `rule_name` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '规则名称（快照）',
  `fingerprint` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT '序列指纹（规则ID+标签集）',
  `instance_ip` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '触发主机IP',
  `labels` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '触发时的标签集',
  `value` decimal(18,4) DEFAULT NULL COMMENT '触发值',
  `threshold` decimal(18,4) DEFAULT NULL COMMENT '阈值（快照）',
  `severity` tinyint DEFAULT '2' COMMENT '级别（1 紧急 2 警告 3 提醒）',
  `status` tinyint DEFAULT '1' COMMENT '状态（1 待确认 2 告警中 3 已恢复）',
  `pending_since` datetime(3) DEFAULT NULL COMMENT '进入待确认时间',
  `fired_at` datetime(3) DEFAULT NULL COMMENT '触发时间',
  `resolved_at` datetime(3) DEFAULT NULL COMMENT '恢复时间',
  `last_notify_at` datetime(3) DEFAULT NULL COMMENT '最近通知时间（冷却判定依据）',
  `notify_count` bigint DEFAULT '0' COMMENT '累计通知次数',
  `suppressed` bigint DEFAULT '0' COMMENT '累计被冷却/静默抑制的次数',
  `summary` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '告警摘要',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `source` varchar(16) COLLATE utf8mb4_general_ci DEFAULT 'monitor' COMMENT '告警来源（monitor 指标 log 日志）',
  PRIMARY KEY (`id`),
  KEY `idx_ops_monitor_alert_event_rule_id` (`rule_id`),
  KEY `idx_ops_monitor_alert_event_fingerprint` (`fingerprint`),
  KEY `idx_ops_monitor_alert_event_status` (`status`),
  KEY `idx_ops_monitor_alert_event_last_notify_at` (`last_notify_at`),
  KEY `idx_ops_monitor_alert_event_source` (`source`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_monitor_notify_channel`;
CREATE TABLE `ops_monitor_notify_channel` (
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '渠道名称',
  `type` tinyint DEFAULT '1' COMMENT '类型（1 Webhook 2 邮件 3 钉钉 4 企业微信 5 飞书）',
  `url` varchar(512) COLLATE utf8mb4_general_ci NOT NULL COMMENT '回调地址（邮件渠道存收件人，逗号分隔）',
  `secret` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'HMAC-SHA256 签名密钥',
  `headers` text COLLATE utf8mb4_general_ci COMMENT '自定义请求头（JSON 对象）',
  `enabled` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '启用状态（1 启用 0 停用）',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_monitor_notify_log`;
CREATE TABLE `ops_monitor_notify_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `event_id` bigint unsigned DEFAULT NULL COMMENT '告警事件ID',
  `rule_id` bigint DEFAULT NULL COMMENT '规则ID',
  `channel_id` bigint DEFAULT NULL COMMENT '渠道ID',
  `channel_name` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '渠道名称（快照）',
  `recovered` tinyint DEFAULT '0' COMMENT '事件类型（0 告警触发 1 告警恢复）',
  `payload` text COLLATE utf8mb4_general_ci COMMENT '请求体',
  `status_code` bigint DEFAULT NULL COMMENT 'HTTP 响应码',
  `response_body` varchar(1024) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '响应体（截断）',
  `attempt` bigint DEFAULT '1' COMMENT '第几次尝试',
  `success` tinyint DEFAULT '0' COMMENT '是否成功（1 成功 0 失败）',
  `error_msg` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '失败原因',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ops_monitor_notify_log_event_id` (`event_id`),
  KEY `idx_ops_monitor_notify_log_rule_id` (`rule_id`),
  KEY `idx_ops_monitor_notify_log_channel_id` (`channel_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_monitor_rule`;
CREATE TABLE `ops_monitor_rule` (
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '规则名称',
  `metric` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '指标名称（展示用）',
  `prom_ql` varchar(512) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'PromQL 表达式，可用 {{instance}} 占位按主机展开',
  `operator` varchar(4) COLLATE utf8mb4_general_ci DEFAULT '>' COMMENT '比较符（> >= < <= == !=），PromQL 未写比较时生效',
  `threshold` decimal(18,4) DEFAULT '0.0000' COMMENT '阈值',
  `severity` tinyint DEFAULT '2' COMMENT '级别（1 紧急 2 警告 3 提醒）',
  `for_sec` int DEFAULT '0' COMMENT '持续时长(秒)后才触发',
  `cooldown_sec` int DEFAULT '300' COMMENT '重复告警冷却(秒)',
  `silence_until` bigint DEFAULT '0' COMMENT '静默截止时间(Unix秒)，0 表示不静默',
  `muted` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '静音（1 只记录事件不通知 0 正常）',
  `channel_ids` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '通知渠道ID列表(逗号分隔)',
  `enabled` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '启用状态（1 启用 0 停用）',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_monitor_target`;
CREATE TABLE `ops_monitor_target` (
  `instance_id` int NOT NULL COMMENT '主机ID',
  `instance_name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '主机名称',
  `instance_ip` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'IP地址',
  `key_id` int DEFAULT '0' COMMENT '采集凭证ID（0 自动选择）',
  `enabled` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '采集开关（1 启用 0 停用）',
  `interval_sec` int DEFAULT '60' COMMENT '采集间隔(秒)',
  `timeout_sec` int DEFAULT '15' COMMENT '采集超时(秒)',
  `last_collect_at` datetime(3) DEFAULT NULL COMMENT '最近采集成功时间',
  `last_status` tinyint DEFAULT '0' COMMENT '采集状态（0 未采集 1 成功 2 失败）',
  `last_error` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '最近采集失败原因',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  KEY `idx_monitor_target_instance` (`instance_id`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_pipeline_steps`;
CREATE TABLE `ops_pipeline_steps` (
  `pipeline_id` bigint NOT NULL COMMENT '编排ID',
  `step_name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '步骤名称',
  `template_id` bigint NOT NULL COMMENT '任务模板ID',
  `step_order` bigint DEFAULT '0' COMMENT '执行顺序',
  `parent_step_id` bigint DEFAULT '0' COMMENT '父步骤ID 0=顶层',
  `condition` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '执行条件',
  `on_failure` tinyint DEFAULT '1' COMMENT '失败策略 1终止 2跳过 3重试',
  `retry_count` bigint DEFAULT '0' COMMENT '重试次数',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  KEY `idx_ops_pipeline_steps_pipeline_id` (`pipeline_id`),
  CONSTRAINT `fk_ops_task_pipelines_steps` FOREIGN KEY (`pipeline_id`) REFERENCES `ops_task_pipelines` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_script`;
CREATE TABLE `ops_script` (
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '脚本名称',
  `content` text COLLATE utf8mb4_general_ci COMMENT '脚本内容',
  `type` varchar(20) COLLATE utf8mb4_general_ci DEFAULT 'bash' COMMENT '脚本类型',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_session_record`;
CREATE TABLE `ops_session_record` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `session_id` varchar(64) NOT NULL COMMENT '会话ID',
  `user_id` int NOT NULL COMMENT '用户ID',
  `instance_id` int NOT NULL COMMENT '主机ID',
  `instance_name` varchar(32) NOT NULL COMMENT '主机名称',
  `instance_ip` varchar(32) NOT NULL COMMENT '主机IP',
  `key_id` int NOT NULL COMMENT '凭证ID',
  `key_name` varchar(32) NOT NULL COMMENT '凭证名称',
  `key_user` varchar(32) NOT NULL COMMENT '登录用户',
  `start_time` datetime NOT NULL COMMENT '开始时间',
  `end_time` datetime DEFAULT NULL COMMENT '结束时间',
  `duration` int DEFAULT '0' COMMENT '会话时长（秒）',
  `status` tinyint(1) DEFAULT '1' COMMENT '状态（1 进行中 2 已结束 3 异常中断）',
  `recording_file` varchar(255) DEFAULT NULL COMMENT '录像文件路径',
  `file_size` bigint DEFAULT '0' COMMENT '文件大小（字节）',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_session_id` (`session_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_instance_id` (`instance_id`),
  KEY `idx_start_time` (`start_time`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=338 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='会话记录表';

DROP TABLE IF EXISTS `ops_step_executions`;
CREATE TABLE `ops_step_executions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `execution_id` bigint unsigned NOT NULL COMMENT '执行记录ID',
  `step_id` bigint NOT NULL COMMENT '编排步骤ID',
  `step_name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '步骤名称',
  `template_id` bigint NOT NULL COMMENT '任务模板ID',
  `status` tinyint DEFAULT '1' COMMENT '状态 1待执行 2执行中 3成功 4失败 5跳过',
  `started_at` datetime(3) DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime(3) DEFAULT NULL COMMENT '结束时间',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ops_step_executions_execution_id` (`execution_id`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_task_executions`;
CREATE TABLE `ops_task_executions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `execution_no` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT '执行编号',
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '执行名称',
  `type` tinyint NOT NULL COMMENT '类型 1快速命令 2快速脚本 3快速文件 4模板执行 5编排执行',
  `source_id` bigint DEFAULT NULL COMMENT '来源ID 模板ID/编排ID',
  `user_id` bigint NOT NULL COMMENT '执行人ID',
  `user_name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '执行人用户名',
  `status` tinyint DEFAULT '1' COMMENT '状态 1待执行 2执行中 3已完成 4部分失败 5全部失败 6已取消',
  `total_hosts` bigint DEFAULT '0' COMMENT '目标主机数',
  `success_hosts` bigint DEFAULT '0' COMMENT '成功主机数',
  `fail_hosts` bigint DEFAULT '0' COMMENT '失败主机数',
  `timeout` bigint DEFAULT '300' COMMENT '超时时间(秒)',
  `started_at` datetime(3) DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime(3) DEFAULT NULL COMMENT '结束时间',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `content` text COLLATE utf8mb4_general_ci COMMENT '执行内容(快速执行时保存)',
  `script_lang` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '脚本语言',
  `src_path` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '源文件路径',
  `dest_path` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '目标文件路径',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ops_task_executions_execution_no` (`execution_no`),
  KEY `idx_ops_task_executions_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_task_pipelines`;
CREATE TABLE `ops_task_pipelines` (
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '编排名称',
  `description` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_task_templates`;
CREATE TABLE `ops_task_templates` (
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '模板名称',
  `type` tinyint NOT NULL COMMENT '任务类型 1执行命令 2执行脚本 3分发文件',
  `content` text COLLATE utf8mb4_general_ci COMMENT '命令内容/脚本内容',
  `script_lang` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '脚本语言 shell/python',
  `src_path` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '源文件路径',
  `dest_path` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '目标路径',
  `timeout` bigint DEFAULT '300' COMMENT '超时时间(秒)',
  `key_id` bigint DEFAULT '0' COMMENT '凭证ID 0自动选择',
  `description` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述',
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_user_instance_auth`;
CREATE TABLE `ops_user_instance_auth` (
  `user_id` int NOT NULL COMMENT '用户ID',
  `instance_id` int DEFAULT NULL COMMENT '主机ID',
  `group_id` int DEFAULT NULL COMMENT '分组ID',
  `auth_type` int NOT NULL COMMENT '类型: 1 主机 2 分组',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `ops_user_instance_key_auth`;
CREATE TABLE `ops_user_instance_key_auth` (
  `user_id` int NOT NULL COMMENT '用户ID',
  `instance_id` int NOT NULL COMMENT '主机ID',
  `key_id` int NOT NULL COMMENT '密钥ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `group_id` int NOT NULL COMMENT '主机分组ID',
  `auth_type` int NOT NULL COMMENT '授权类型（1 主机 2 主机组）'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_dept`;
CREATE TABLE `sys_dept` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '部门名称',
  `parent_id` bigint DEFAULT NULL COMMENT '父级ID',
  `order_num` int DEFAULT NULL COMMENT '排序',
  `status` tinyint(1) DEFAULT '0' COMMENT '状态（true正常 false停用）',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_sys_dept_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_log`;
CREATE TABLE `sys_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `method` varchar(8) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '请求方法',
  `request_uri` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '请求地址',
  `params` text COLLATE utf8mb4_general_ci COMMENT '请求参数',
  `resp` text COLLATE utf8mb4_general_ci COMMENT 'resp',
  `ip_addr` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '请求IP',
  `cost_time` bigint DEFAULT NULL COMMENT '响应耗时',
  `status_code` char(8) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '响应状态码',
  `cost_time_ms` bigint DEFAULT NULL COMMENT '响应耗时',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  `dept_id` bigint DEFAULT NULL COMMENT '创建人部门',
  `create_user` varchar(16) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人用户名',
  `oper_module` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '操作模块',
  `oper_type` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '操作类型（query/add/edit/delete/grant/other）',
  `sensitive` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '敏感操作（1 是 0 否）',
  `biz_code` varchar(16) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '业务返回码',
  `biz_msg` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '业务返回信息',
  `user_agent` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '客户端UA',
  `operate_at` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '操作时间(格式化)',
  PRIMARY KEY (`id`),
  KEY `idx_sys_log_oper_module` (`oper_module`),
  KEY `idx_sys_log_oper_type` (`oper_type`),
  KEY `idx_sys_log_sensitive` (`sensitive`)
) ENGINE=InnoDB AUTO_INCREMENT=63208 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_login_log`;
CREATE TABLE `sys_login_log` (
  `user_name` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '登录用户名',
  `user_id` int DEFAULT '0' COMMENT '用户ID（用户不存在时为0）',
  `result` tinyint DEFAULT '0' COMMENT '结果（0 失败 1 成功 2 登出）',
  `reason` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '失败原因',
  `ip_addr` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '来源IP',
  `location` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '来源IP归属（预留）',
  `user_agent` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '客户端UA',
  `browser` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '浏览器',
  `os` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '操作系统',
  `platform` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '登录端（web/bastion）',
  `dept_id` int DEFAULT '0' COMMENT '部门ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '登录时间',
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `idx_sys_login_log_user_name` (`user_name`),
  KEY `idx_sys_login_log_result` (`result`),
  KEY `idx_sys_login_log_ip_addr` (`ip_addr`),
  KEY `idx_sys_login_log_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=37714 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_menu`;
CREATE TABLE `sys_menu` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT '菜单名称',
  `parent_id` int DEFAULT NULL COMMENT '父菜单ID',
  `order_num` int DEFAULT NULL COMMENT '显示顺序',
  `path` varchar(200) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '路由地址',
  `component` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '组件路径',
  `keep_alive` tinyint(1) DEFAULT NULL COMMENT '是否缓存（true缓存 false不缓存）',
  `is_affix` tinyint(1) DEFAULT NULL COMMENT '是否固定（1 是 0 否）',
  `is_iframe` tinyint(1) DEFAULT NULL COMMENT '是否为内链（1是 0否）',
  `is_hide` tinyint(1) DEFAULT NULL COMMENT '是否隐藏（1隐藏 0显示）',
  `is_link` tinyint(1) DEFAULT NULL COMMENT '是否为外链（1是 0否）',
  `url` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '外链地址',
  `type` char(1) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '菜单类型（M目录 C菜单 F按钮）',
  `status` char(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '菜单状态（1正常 0停用）',
  `perms` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '权限标识',
  `icon` varchar(100) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '菜单图标',
  `request_url` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '接口请求地址',
  `request_method` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '接口请求方法',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_sys_menu_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=160 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_role`;
CREATE TABLE `sys_role` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '角色名称',
  `order_num` int DEFAULT '1' COMMENT '排序',
  `status` varchar(1) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '1' COMMENT '状态（0：禁用，1：正常）',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=32 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_role_menu`;
CREATE TABLE `sys_role_menu` (
  `role_id` int NOT NULL,
  `menu_id` int NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_user`;
CREATE TABLE `sys_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `dept_id` bigint NOT NULL COMMENT '部门ID',
  `user_name` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '用户名',
  `nick_name` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '昵称',
  `email` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '邮箱',
  `phone` varchar(16) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '电话',
  `sex` tinyint DEFAULT NULL COMMENT '性别（0男 1女 2未知）',
  `avatar` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '头像地址',
  `password` varchar(512) COLLATE utf8mb4_general_ci NOT NULL COMMENT '密码',
  `status` varchar(1) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '帐号状态（1正常 0停用）',
  `login_ip` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '登录IP',
  `login_date` datetime(3) DEFAULT NULL COMMENT '登录时间',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `create_by` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人ID',
  `update_by` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '修改人ID',
  `del_flag` varchar(1) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '删除标识（0正常 1 已删除）',
  `remark` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '备注',
  `totp_secret` varchar(512) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'TOTP密钥（加密存储）',
  `totp_enabled` tinyint DEFAULT '0' COMMENT 'MFA状态（0未绑定 1已启用）',
  `recovery_codes` text COLLATE utf8mb4_general_ci COMMENT '一次性恢复码（bcrypt哈希JSON数组）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_sys_user_user_name` (`user_name`)
) ENGINE=InnoDB AUTO_INCREMENT=100 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_user_role`;
CREATE TABLE `sys_user_role` (
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `role_id` bigint NOT NULL COMMENT '角色ID'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `sys_user_token`;
CREATE TABLE `sys_user_token` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `token` longtext COLLATE utf8mb4_general_ci,
  `created_at` datetime(3) DEFAULT NULL,
  `expire_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=137 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

