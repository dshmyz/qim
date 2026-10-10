-- =====================================================================
-- 清理重复的未完成任务（存量数据修复）
-- 数据库：MySQL（8.0+）/ SQLite（两版各一段，任选其一执行）
-- 背景：TodoExtractor 历史上落库零幂等——AI 提取对同一条待办重复触发（单次提取
--       重复返回、或同一待办在多条消息里被反复提到）都会再插一行，任务表堆出
--       同名+同截止日期的重复行。list_tasks 把它们原样吐给模型，引发模型反复
--       对账、重排、自我纠正（详见 2026-10 悬浮球 AI 助手格式混乱排查）。
-- 判重口径：user_id + title + due_date（NULL 视为相同），仅针对未完成任务
--       （status != 'done'）；已完成记录保留不动。优先级差异不算重复——同一
--       待办被提取出不同优先级仍是重复。
-- 幂等：保留每组最早一条（id 最小），其余删除；重复执行结果不变。
-- 关联修复：TodoExtractor 落库幂等 + ListTasksTool 出口去重已随本脚本同批上线，
--           新增数据不会再产生此形态的重复。
-- =====================================================================

-- ---------------------------------------------------------------------
-- 第 0 步：备份（建议）
-- 生产环境务必先备份，或先在测试/灰度库执行。
-- ---------------------------------------------------------------------
-- CREATE TABLE tasks_bak_YYYYMMDD AS SELECT * FROM tasks;

-- ---------------------------------------------------------------------
-- 第 1 步：巡检——有多少组重复、共涉及多少行
-- ---------------------------------------------------------------------
-- MySQL / SQLite 通用：
-- SELECT user_id, title, due_date, COUNT(*) AS cnt
-- FROM tasks
-- WHERE status != 'done' AND deleted_at IS NULL
-- GROUP BY user_id, title, due_date
-- HAVING COUNT(*) > 1;

-- ---------------------------------------------------------------------
-- 第 2 步：清理（MySQL 8.0+）
-- <=> 为 NULL 安全等号：due_date 双方都 NULL 视为相等。
-- ---------------------------------------------------------------------
DELETE t1 FROM tasks t1
JOIN tasks t2
  ON  t1.user_id = t2.user_id
  AND t1.title   = t2.title
  AND (t1.due_date <=> t2.due_date)
  AND t2.status <> 'done'
  AND t2.deleted_at IS NULL
  AND t2.id < t1.id
WHERE t1.status <> 'done'
  AND t1.deleted_at IS NULL;

-- ---------------------------------------------------------------------
-- 第 2 步：清理（SQLite）
-- 子查询只统计未完成行、取每组最小 id 保留；已完成任务不参与也不被删。
-- ---------------------------------------------------------------------
-- DELETE FROM tasks
-- WHERE status != 'done'
--   AND deleted_at IS NULL
--   AND id NOT IN (
--     SELECT MIN(id) FROM tasks
--     WHERE status != 'done' AND deleted_at IS NULL
--     GROUP BY user_id, title, IFNULL(due_date, '')
--   );

-- ---------------------------------------------------------------------
-- 第 3 步：复核——重复组应归零
-- ---------------------------------------------------------------------
-- SELECT user_id, title, due_date, COUNT(*) AS cnt
-- FROM tasks
-- WHERE status != 'done' AND deleted_at IS NULL
-- GROUP BY user_id, title, due_date
-- HAVING COUNT(*) > 1;
