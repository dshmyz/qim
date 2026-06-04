-- ============================================
-- OIM Server 数据库建表语句
-- 数据库类型：MySQL
-- 生成时间：2026-04-28
-- 说明：所有继承 PrimaryStringEntity 的实体都包含公共字段
-- 公共字段模板：
--   id              VARCHAR(40)    主键ID
--   createdTimestamp BIGINT        创建时间戳（毫秒）
--   updatedTimestamp BIGINT        更新时间戳（毫秒）
--   isDeleted       INT           是否删除 0:否 1:是
--   createdDateTime VARCHAR(32)    创建时间
--   updatedDateTime VARCHAR(32)    更新时间
-- ============================================

-- 设置字符集
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================
-- 1. 用户核心模块
-- ============================================

-- 用户主表
DROP TABLE IF EXISTS `w_user`;
CREATE TABLE `w_user` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `number` BIGINT NOT NULL DEFAULT 0 COMMENT '用户数字账号',
  `account` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '账号',
  `email` VARCHAR(150) NOT NULL DEFAULT '' COMMENT '电子邮箱',
  `mobile` VARCHAR(30) NOT NULL DEFAULT '' COMMENT '手机号码',
  `password` VARCHAR(250) NOT NULL DEFAULT '' COMMENT '密码',
  `head` VARCHAR(4) NOT NULL DEFAULT '1' COMMENT '系统头像编号',
  `avatar` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '自定义照片',
  `nickname` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '昵称',
  `name` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '真实姓名',
  `spell` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '姓名拼写',
  `simpleSpell` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '姓名简拼',
  `gender` VARCHAR(2) NOT NULL DEFAULT '3' COMMENT '性别 1:男 2:女 3:保密',
  `telephone` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '联系电话',
  `homeAddress` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '家庭地址',
  `locationAddress` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '所在地址',
  `homeZipCode` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '家庭邮政编码',
  `locationZipCode` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '所在地邮政编码',
  `locationCountryCode` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '所在地国家编码',
  `locationProvinceCode` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '所在地州省编码',
  `locationCityCode` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '所在地城市编码',
  `constellationCode` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '星座',
  `blood` VARCHAR(5) NOT NULL DEFAULT '' COMMENT '血型',
  `birthDate` VARCHAR(32) NOT NULL DEFAULT '0001-01-01' COMMENT '出生日期',
  `age` INT(4) NOT NULL DEFAULT 0 COMMENT '年龄',
  `identityCard` VARCHAR(30) NOT NULL DEFAULT '' COMMENT '身份证号码',
  `maritalStatus` VARCHAR(5) NOT NULL DEFAULT '' COMMENT '婚姻状况 1:未婚 2:已婚',
  `nativePlace` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '籍贯',
  `nationName` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '民族',
  `politicsStatus` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '政治面貌',
  `introduce` VARCHAR(1500) NOT NULL DEFAULT '' COMMENT '介绍',
  `remark` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '备注',
  `qq` VARCHAR(30) NOT NULL DEFAULT '' COMMENT 'QQ号码',
  `education` VARCHAR(30) NOT NULL DEFAULT '' COMMENT '学历',
  `school` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '毕业学校',
  `professional` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '专业',
  `graduationDate` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '毕业时间',
  `workDate` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '参加工作时间',
  `signature` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '个性签名',
  `isDisable` INT NOT NULL DEFAULT 0 COMMENT '是否禁用 0:启用 1:停用',
  `canceledTimestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '注销时间',
  `type` VARCHAR(2) NOT NULL DEFAULT '0' COMMENT '用户类型 0:普通 1:管理员 2:超级管理员',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `index_number` (`number`),
  UNIQUE INDEX `index_account` (`account`),
  INDEX `index_email` (`email`),
  INDEX `index_mobile` (`mobile`),
  INDEX `index_nickname` (`nickname`),
  INDEX `index_name` (`name`),
  INDEX `index_simpleSpell` (`simpleSpell`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户主表';

-- 用户数字账号生成器（不继承PrimaryStringEntity，使用自增Long主键）
DROP TABLE IF EXISTS `w_user_number`;
CREATE TABLE `w_user_number` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `createdDateTime` VARCHAR(32) NOT NULL DEFAULT '0001-01-01 00:00:00' COMMENT '创建时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户数字账号生成器';

-- 用户头像设置
DROP TABLE IF EXISTS `w_user_head`;
CREATE TABLE `w_user_head` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `headId` VARCHAR(40) NOT NULL DEFAULT '1' COMMENT '系统头像编码',
  `fileName` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '自定义头像文件名',
  `type` VARCHAR(2) NOT NULL DEFAULT '1' COMMENT '头像类型 1:系统头像 2:自定义头像',
  `url` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '自定义头像地址',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户头像设置';

-- 用户密保问题
DROP TABLE IF EXISTS `w_user_security_question`;
CREATE TABLE `w_user_security_question` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `question` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '密保问题',
  `answer` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '密保答案',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户密保问题';

-- ============================================
-- 2. 权限管理模块（RBAC）
-- ============================================

-- 角色表
DROP TABLE IF EXISTS `m_role`;
CREATE TABLE `m_role` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '角色名字',
  `introduce` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '角色描述',
  `isDisable` INT NOT NULL DEFAULT 0 COMMENT '是否禁用 0:启用 1:停用',
  `grade` INT NOT NULL DEFAULT 1 COMMENT '角色级别 0:系统角色 1:普通角色',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色表';

-- 功能菜单表
DROP TABLE IF EXISTS `m_function`;
CREATE TABLE `m_function` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `service` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '服务名',
  `code` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '菜单编码',
  `superId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '上级菜单ID',
  `name` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '菜单名称',
  `isDisable` INT NOT NULL DEFAULT 0 COMMENT '是否禁用 0:启用 1:停用',
  `introduce` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '功能描述',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '菜单排序',
  `path` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '功能地址',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='功能菜单表';

-- 用户角色关联表
DROP TABLE IF EXISTS `m_user_role`;
CREATE TABLE `m_user_role` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `roleId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '角色ID',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`),
  INDEX `index_roleId` (`roleId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色关联表';

-- 角色功能关联表
DROP TABLE IF EXISTS `m_role_function`;
CREATE TABLE `m_role_function` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `roleId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '角色ID',
  `functionId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '功能ID',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_roleId` (`roleId`),
  INDEX `index_functionId` (`functionId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色功能关联表';

-- ============================================
-- 3. 群组模块
-- ============================================

-- 群组主表
DROP TABLE IF EXISTS `w_group`;
CREATE TABLE `w_group` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `number` BIGINT NOT NULL DEFAULT 0 COMMENT '群号码',
  `name` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '群名称',
  `head` VARCHAR(5) NOT NULL DEFAULT '' COMMENT '系统头像',
  `avatar` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '自定义头像',
  `classification` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '分类',
  `introduce` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '介绍',
  `location` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '位置',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `index_number` (`number`),
  INDEX `index_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群组主表';

-- 群号码生成器（不继承PrimaryStringEntity，使用自增Long主键）
DROP TABLE IF EXISTS `w_group_number`;
CREATE TABLE `w_group_number` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `createdDateTime` VARCHAR(32) NOT NULL DEFAULT '0001-01-01 00:00:00' COMMENT '创建时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群号码生成器';

-- 群成员表
DROP TABLE IF EXISTS `w_group_member`;
CREATE TABLE `w_group_member` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `position` VARCHAR(2) NOT NULL DEFAULT '3' COMMENT '权限 1:群主 2:管理员 3:普通成员',
  `nickname` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '群备注名',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_groupId` (`groupId`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群成员表';

-- 入群设置表
DROP TABLE IF EXISTS `w_group_join_setting`;
CREATE TABLE `w_group_join_setting` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `joinType` VARCHAR(2) NOT NULL DEFAULT '2' COMMENT '申请加群方式 1:允许任何人 2:需要验证 3:回答问题 4:问题+审核 5:仅邀请 6:不允许',
  `inviteType` VARCHAR(2) NOT NULL DEFAULT '4' COMMENT '邀请加群方式 1:不允许 2:管理员邀请 3:任何人邀请 4:需要验证',
  `question` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '入群问题',
  `answer` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '入群答案',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `index_groupId` (`groupId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='入群设置表';

-- 入群申请表
DROP TABLE IF EXISTS `w_group_join_apply`;
CREATE TABLE `w_group_join_apply` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `applyUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '请求用户ID',
  `categoryId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '请求用户设定的群分组ID',
  `remark` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '请求用户设定的群备注名',
  `applyMessage` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '附加消息',
  `verifyType` VARCHAR(2) NOT NULL DEFAULT '2' COMMENT '当时设定的验证方式',
  `question` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '验证问题',
  `answer` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '验证答案',
  `handleType` VARCHAR(2) NOT NULL DEFAULT '0' COMMENT '处理结果 0:未处理 1:同意 2:拒绝 3:忽略',
  `handleTimestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '处理时间(毫秒)',
  `handleUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '处理验证用户的ID(群主/管理员)',
  `handleUserPosition` VARCHAR(2) NOT NULL DEFAULT '1' COMMENT '处理验证用户的权限(群主/管理员)',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_applyUserId` (`applyUserId`),
  INDEX `index_groupId` (`groupId`),
  INDEX `index_handleUserId` (`handleUserId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='入群申请表';

-- 群分组表
DROP TABLE IF EXISTS `w_group_category`;
CREATE TABLE `w_group_category` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '分组名称',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序',
  `type` INT NOT NULL DEFAULT 2 COMMENT '类型 1:系统默认 2:用户自定义',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群分组表';

-- 群头像表
DROP TABLE IF EXISTS `w_group_head`;
CREATE TABLE `w_group_head` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `rootPath` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '根目录',
  `nodePath` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '节点目录',
  `fullPath` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整路径(不含文件名)',
  `saveName` VARCHAR(250) NOT NULL DEFAULT '' COMMENT '保存文件名(不含后缀)',
  `saveFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '保存文件名(含后缀)',
  `fullPathName` VARCHAR(850) NOT NULL DEFAULT '' COMMENT '完整路径名',
  `originalName` VARCHAR(280) NOT NULL DEFAULT '' COMMENT '原文件名(不含后缀)',
  `originalFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '原文件名(含后缀)',
  `extension` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '后缀名',
  `url` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整下载URL',
  `type` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '文件类型',
  `size` BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
  `md5` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件MD5',
  `sha1` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件SHA1',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '上传用户ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '上传群ID',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_groupId` (`groupId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群头像表';

-- 群关系表
DROP TABLE IF EXISTS `w_group_relation`;
CREATE TABLE `w_group_relation` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `categoryId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群分组ID',
  `remark` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '群备注',
  `isBlocked` VARCHAR(2) NOT NULL DEFAULT '0' COMMENT '是否屏蔽 0:否 1:是',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`),
  INDEX `index_groupId` (`groupId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群关系表';

-- 群公告表
DROP TABLE IF EXISTS `w_group_notice`;
CREATE TABLE `w_group_notice` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '发布人ID',
  `content` TEXT COMMENT '公告内容',
  `timestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '发布时间',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_groupId` (`groupId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群公告表';

-- 邀请入群申请表
DROP TABLE IF EXISTS `w_group_invite_apply`;
CREATE TABLE `w_group_invite_apply` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `inviteUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '邀请人ID',
  `targetUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '被邀请人ID',
  `handleType` VARCHAR(2) NOT NULL DEFAULT '0' COMMENT '处理结果 0:未处理 1:同意 2:拒绝 3:忽略',
  `handleTimestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '处理时间',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_groupId` (`groupId`),
  INDEX `index_inviteUserId` (`inviteUserId`),
  INDEX `index_targetUserId` (`targetUserId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邀请入群申请表';

-- 入群验证问题表
DROP TABLE IF EXISTS `w_group_join_verify_question`;
CREATE TABLE `w_group_join_verify_question` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `question` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '问题',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_groupId` (`groupId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='入群验证问题表';

-- 入群验证答案表
DROP TABLE IF EXISTS `w_group_join_verify_answer`;
CREATE TABLE `w_group_join_verify_answer` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `questionId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '问题ID',
  `answer` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '答案',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_groupId` (`groupId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='入群验证答案表';

-- ============================================
-- 4. 聊天模块
-- ============================================

-- 单聊消息记录表
DROP TABLE IF EXISTS `im_user_chat_content`;
CREATE TABLE `im_user_chat_content` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `ownKey` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '个人消息key',
  `messageKey` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '消息key',
  `sendUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '发送者ID',
  `sendUserHead` VARCHAR(5) NOT NULL DEFAULT '1' COMMENT '发送者系统头像',
  `sendUserAvatar` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '发送者自定义头像',
  `sendUserName` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '发送者姓名',
  `sendUserNickname` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '发送者昵称',
  `sendUserRemarkName` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '发送者备注名',
  `receiveUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '接收者ID',
  `receiveUserHead` VARCHAR(5) NOT NULL DEFAULT '' COMMENT '接收者系统头像',
  `receiveUserAvatar` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '接收者自定义头像',
  `receiveUserName` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '接收者姓名',
  `receiveUserNickname` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '接收者昵称',
  `receiveUserRemarkName` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '接收者备注名',
  `isSend` VARCHAR(2) DEFAULT '0' COMMENT '是否已发送 0:否 1:是',
  `isRead` VARCHAR(2) DEFAULT '0' COMMENT '是否已读 0:否 1:是',
  `isDeleted` VARCHAR(2) DEFAULT '0' COMMENT '是否删除 0:否 1:是',
  `dateTime` DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00' COMMENT '时间',
  `timestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '时间戳',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_messageKey` (`messageKey`),
  INDEX `index_receiveUserId` (`receiveUserId`),
  INDEX `index_sendUserId` (`sendUserId`),
  INDEX `index_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='单聊消息记录表';

-- 群聊消息记录表
DROP TABLE IF EXISTS `im_group_chat_content`;
CREATE TABLE `im_group_chat_content` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `messageKey` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '消息key',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `groupName` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '群当时名称',
  `groupHead` VARCHAR(5) NOT NULL DEFAULT '1' COMMENT '群系统头像',
  `groupAvatar` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '群自定义头像',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '发送者ID',
  `userHead` VARCHAR(5) NOT NULL DEFAULT '1' COMMENT '用户系统头像',
  `userAvatar` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '用户自定义头像',
  `userPosition` VARCHAR(2) NOT NULL DEFAULT '3' COMMENT '用户权限',
  `userName` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '用户姓名',
  `userNickname` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '用户昵称',
  `userGroupNickname` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '群内昵称',
  `dateTime` DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00' COMMENT '时间',
  `timestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '时间戳',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` VARCHAR(2) DEFAULT '0' COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_messageKey` (`messageKey`),
  INDEX `index_groupId` (`groupId`),
  INDEX `index_userId` (`userId`),
  INDEX `index_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群聊消息记录表';

-- 最近会话表
DROP TABLE IF EXISTS `im_recent_chat`;
CREATE TABLE `im_recent_chat` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `messageKey` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '消息key',
  `contentId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '消息ID',
  `chatId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '聊天对象ID',
  `type` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '类型 1:联系人 2:群 3:讨论组 4:聊天室 5:会议',
  `dateTime` DATETIME NOT NULL COMMENT '时间',
  `timestamp` BIGINT NOT NULL COMMENT '时间戳',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`),
  INDEX `index_messageKey` (`messageKey`),
  INDEX `index_contentId` (`contentId`),
  INDEX `index_chatId` (`chatId`),
  INDEX `index_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='最近会话表';

-- 单聊未读计数表
DROP TABLE IF EXISTS `im_user_chat_unread`;
CREATE TABLE `im_user_chat_unread` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `receiveUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '接收者ID',
  `sendUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '发送者ID',
  `lastContentId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '最后一条消息ID',
  `unreadCount` INT(11) NOT NULL DEFAULT 0 COMMENT '未读数量',
  `timestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '时间戳',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_receiveUserId` (`receiveUserId`),
  INDEX `index_sendUserId` (`sendUserId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='单聊未读计数表';

-- 群聊未读计数表
DROP TABLE IF EXISTS `im_group_chat_unread`;
CREATE TABLE `im_group_chat_unread` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '收到消息用户ID',
  `lastContentId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '最后一条消息ID',
  `unreadCount` INT(11) NOT NULL DEFAULT 0 COMMENT '未读数量',
  `timestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '时间戳',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_groupId` (`groupId`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群聊未读计数表';

-- 单聊消息内容项表
DROP TABLE IF EXISTS `im_user_chat_item`;
CREATE TABLE `im_user_chat_item` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `contentId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '消息ID(关联im_user_chat_content.id)',
  `ownKey` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '个人消息key',
  `messageKey` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '消息key',
  `receiveUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '接收者ID',
  `sendUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '发送者ID',
  `type` VARCHAR(5) NOT NULL DEFAULT '' COMMENT '类型: text文本 face表情 url链接 image图片 file文件 audio音频 video视频 position位置 at提醒',
  `originalValue` TEXT COMMENT '原始内容',
  `filterValue` TEXT COMMENT '过滤后内容(敏感词过滤)',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序',
  `section` INT NOT NULL DEFAULT 0 COMMENT '段落',
  `timestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '时间戳(毫秒)',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_contentId` (`contentId`),
  INDEX `index_messageKey` (`messageKey`),
  INDEX `index_receiveUserId` (`receiveUserId`),
  INDEX `index_sendUserId` (`sendUserId`),
  INDEX `index_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='单聊消息内容项表';

-- 群聊消息内容项表
DROP TABLE IF EXISTS `im_group_chat_item`;
CREATE TABLE `im_group_chat_item` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `contentId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '消息ID(关联im_group_chat_content.id)',
  `messageKey` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '消息key',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '群ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '发送消息用户ID',
  `type` VARCHAR(5) NOT NULL DEFAULT '' COMMENT '类型: text文本 face表情 url链接 image图片 file文件 audio音频 video视频 position位置 at提醒',
  `originalValue` TEXT COMMENT '原始内容',
  `filterValue` TEXT COMMENT '过滤后内容(敏感词过滤)',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序',
  `section` INT NOT NULL DEFAULT 0 COMMENT '段落',
  `timestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '时间戳(毫秒)',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_contentId` (`contentId`),
  INDEX `index_messageKey` (`messageKey`),
  INDEX `index_groupId` (`groupId`),
  INDEX `index_userId` (`userId`),
  INDEX `index_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群聊消息内容项表';

-- ============================================
-- 5. 联系人模块
-- ============================================

-- 联系人关系表
DROP TABLE IF EXISTS `w_contact_relation`;
CREATE TABLE `w_contact_relation` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `ownerUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '所属用户ID',
  `categoryId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '联系人分组ID',
  `contactUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '联系人用户ID',
  `remark` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '联系人备注名',
  `isBlocked` VARCHAR(2) NOT NULL DEFAULT '0' COMMENT '是否加入黑名单 0:否 1:是',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_ownerUserId` (`ownerUserId`),
  INDEX `index_categoryId` (`categoryId`),
  INDEX `index_contactUserId` (`contactUserId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='联系人关系表';

-- 联系人分组表
DROP TABLE IF EXISTS `w_contact_category`;
CREATE TABLE `w_contact_category` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '所属用户ID',
  `name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '分组名字',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序',
  `type` INT NOT NULL DEFAULT 2 COMMENT '类型 1:系统默认 2:用户自定义',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='联系人分组表';

-- 添加好友申请表
DROP TABLE IF EXISTS `w_contact_add_apply`;
CREATE TABLE `w_contact_add_apply` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `applyUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '请求用户ID',
  `targetUserId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '被添加的用户ID',
  `categoryId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '分组ID',
  `remark` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '备注名',
  `verifyType` VARCHAR(40) NOT NULL DEFAULT '2' COMMENT '验证方式',
  `question` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '问题',
  `answer` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '答案',
  `handleType` VARCHAR(2) NOT NULL DEFAULT '' COMMENT '处理结果 0:未处理 1:同意 2:拒绝 3:忽略',
  `handleTimestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '处理时间',
  `applyMessage` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '附加消息',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_applyUserId` (`applyUserId`),
  INDEX `index_targetUserId` (`targetUserId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='添加好友申请表';

-- 好友验证问题表
DROP TABLE IF EXISTS `w_contact_add_verify_question`;
CREATE TABLE `w_contact_add_verify_question` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `question` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '问题',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='好友验证问题表';

-- 好友验证答案表
DROP TABLE IF EXISTS `w_contact_add_verify_answer`;
CREATE TABLE `w_contact_add_verify_answer` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `questionId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '问题ID',
  `answer` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '答案',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='好友验证答案表';

-- 防骚扰设置表
DROP TABLE IF EXISTS `w_contact_harass_setting`;
CREATE TABLE `w_contact_harass_setting` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '用户ID',
  `allowStrangerChat` VARCHAR(2) NOT NULL DEFAULT '1' COMMENT '允许陌生人聊天 0:否 1:是',
  `allowStrangerFind` VARCHAR(2) NOT NULL DEFAULT '1' COMMENT '允许陌生人找到我 0:否 1:是',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='防骚扰设置表';

-- ============================================
-- 6. 文件模块
-- ============================================

-- 通用文件表
DROP TABLE IF EXISTS `base_file_data`;
CREATE TABLE `base_file_data` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `rootPath` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '根目录',
  `nodePath` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '节点目录',
  `fullPath` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整路径(不含文件名)',
  `saveName` VARCHAR(250) NOT NULL DEFAULT '' COMMENT '保存文件名(不含后缀)',
  `saveFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '保存文件名(含后缀)',
  `fullPathName` VARCHAR(850) NOT NULL DEFAULT '' COMMENT '完整路径名',
  `originalName` VARCHAR(280) NOT NULL DEFAULT '' COMMENT '原文件名(不含后缀)',
  `originalFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '原文件名(含后缀)',
  `extension` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '后缀名',
  `url` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整下载URL',
  `type` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '文件类型',
  `size` BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
  `md5` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件MD5',
  `sha1` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件SHA1',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '上传用户ID',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`),
  INDEX `index_md5` (`md5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通用文件表';

-- 图片表
DROP TABLE IF EXISTS `base_image_data`;
CREATE TABLE `base_image_data` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `rootPath` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '根目录',
  `nodePath` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '节点目录',
  `fullPath` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整路径(不含文件名)',
  `saveName` VARCHAR(250) NOT NULL DEFAULT '' COMMENT '保存文件名(不含后缀)',
  `saveFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '保存文件名(含后缀)',
  `fullPathName` VARCHAR(850) NOT NULL DEFAULT '' COMMENT '完整路径名',
  `originalName` VARCHAR(280) NOT NULL DEFAULT '' COMMENT '原文件名(不含后缀)',
  `originalFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '原文件名(含后缀)',
  `extension` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '后缀名',
  `url` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整下载URL',
  `type` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '文件类型',
  `size` BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
  `md5` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件MD5',
  `sha1` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件SHA1',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '上传用户ID',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`),
  INDEX `index_md5` (`md5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图片表';

-- 用户头像文件表
DROP TABLE IF EXISTS `base_user_head_data`;
CREATE TABLE `base_user_head_data` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `rootPath` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '根目录',
  `nodePath` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '节点目录',
  `fullPath` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整路径(不含文件名)',
  `saveName` VARCHAR(250) NOT NULL DEFAULT '' COMMENT '保存文件名(不含后缀)',
  `saveFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '保存文件名(含后缀)',
  `fullPathName` VARCHAR(850) NOT NULL DEFAULT '' COMMENT '完整路径名',
  `originalName` VARCHAR(280) NOT NULL DEFAULT '' COMMENT '原文件名(不含后缀)',
  `originalFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '原文件名(含后缀)',
  `extension` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '后缀名',
  `url` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整下载URL',
  `type` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '文件类型',
  `size` BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
  `md5` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件MD5',
  `sha1` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件SHA1',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '上传用户ID',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户头像文件表';

-- 群头像文件表
DROP TABLE IF EXISTS `base_group_head_data`;
CREATE TABLE `base_group_head_data` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `rootPath` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '根目录',
  `nodePath` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '节点目录',
  `fullPath` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整路径(不含文件名)',
  `saveName` VARCHAR(250) NOT NULL DEFAULT '' COMMENT '保存文件名(不含后缀)',
  `saveFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '保存文件名(含后缀)',
  `fullPathName` VARCHAR(850) NOT NULL DEFAULT '' COMMENT '完整路径名',
  `originalName` VARCHAR(280) NOT NULL DEFAULT '' COMMENT '原文件名(不含后缀)',
  `originalFullName` VARCHAR(300) NOT NULL DEFAULT '' COMMENT '原文件名(含后缀)',
  `extension` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '后缀名',
  `url` VARCHAR(800) NOT NULL DEFAULT '' COMMENT '完整下载URL',
  `type` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '文件类型',
  `size` BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
  `md5` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件MD5',
  `sha1` VARCHAR(80) NOT NULL DEFAULT '' COMMENT '文件SHA1',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '上传用户ID',
  `groupId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '上传群ID',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`),
  INDEX `index_groupId` (`groupId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群头像文件表';

-- ============================================
-- 7. 系统设置模块
-- ============================================

-- 客户端版本表
DROP TABLE IF EXISTS `setting_app_client`;
CREATE TABLE `setting_app_client` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '客户端名称',
  `build` INT NOT NULL DEFAULT 1 COMMENT '内部构建版本',
  `version` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '发布版本号',
  `type` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '客户端类型 1:JavaFX 2:Electron 3:H5 4:Android 5:iOS',
  `platform` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '客户端平台',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_build` (`build`),
  INDEX `index_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='客户端版本表';

-- 多端在线策略表
DROP TABLE IF EXISTS `setting_multiple_online_strategy`;
CREATE TABLE `setting_multiple_online_strategy` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `strategy` VARCHAR(2) NOT NULL DEFAULT '1' COMMENT '策略 1:允许 2:不允许',
  `description` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '描述',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='多端在线策略表';

-- ============================================
-- 8. 通知模块
-- ============================================

-- 文本通知表
DROP TABLE IF EXISTS `w_text_notice`;
CREATE TABLE `w_text_notice` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `url` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '链接地址',
  `title` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '标题',
  `content` TEXT COMMENT '内容',
  `timestamp` BIGINT NOT NULL DEFAULT 0 COMMENT '时间戳',
  `openType` VARCHAR(2) NOT NULL DEFAULT '' COMMENT '打开方式 1:app 2:browser',
  `pushType` VARCHAR(2) NOT NULL DEFAULT '' COMMENT '推送类型 1:所有用户 2:指定用户',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文本通知表';

-- 用户通知记录表
DROP TABLE IF EXISTS `w_user_text_notice`;
CREATE TABLE `w_user_text_notice` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `userId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '接收通知的用户ID',
  `textNoticeId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '通知表ID',
  `isRead` VARCHAR(2) NOT NULL DEFAULT '0' COMMENT '是否已读 0:未读 1:已读',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_userId` (`userId`),
  INDEX `index_textNoticeId` (`textNoticeId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户通知记录表';

-- ============================================
-- 9. 其他模块
-- ============================================

-- 敏感词过滤表
DROP TABLE IF EXISTS `im_words_filter`;
CREATE TABLE `im_words_filter` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `words` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '敏感词',
  `level` INT NOT NULL DEFAULT 0 COMMENT '级别',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='敏感词过滤表';

-- 服务端动作信息表
DROP TABLE IF EXISTS `server_action_info`;
CREATE TABLE `server_action_info` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `service` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '服务名',
  `action` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '动作名',
  `path` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '路径',
  `method` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '请求方法',
  `description` VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '描述',
  `isEnable` VARCHAR(2) NOT NULL DEFAULT '1' COMMENT '是否启用 0:否 1:是',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_service` (`service`),
  INDEX `index_action` (`action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务端动作信息表';

-- 服务器类型表
DROP TABLE IF EXISTS `server_type`;
CREATE TABLE `server_type` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `code` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '类型编码',
  `name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '类型名称',
  `description` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '描述',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务器类型表';

-- 服务器地址表
DROP TABLE IF EXISTS `server_address`;
CREATE TABLE `server_address` (
  `id` VARCHAR(40) NOT NULL COMMENT '主键ID',
  `typeId` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '服务器类型ID',
  `host` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '主机地址',
  `port` INT NOT NULL DEFAULT 0 COMMENT '端口',
  `protocol` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '协议',
  `isEnable` VARCHAR(2) NOT NULL DEFAULT '1' COMMENT '是否启用 0:否 1:是',
  `weight` INT NOT NULL DEFAULT 1 COMMENT '权重',
  `createdTimestamp` BIGINT COMMENT '创建时间戳',
  `updatedTimestamp` BIGINT COMMENT '更新时间戳',
  `isDeleted` INT DEFAULT 0 COMMENT '是否删除 0:否 1:是',
  `createdDateTime` VARCHAR(32) COMMENT '创建时间',
  `updatedDateTime` VARCHAR(32) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `index_typeId` (`typeId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务器地址表';

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================
-- 建表完成
-- ============================================
