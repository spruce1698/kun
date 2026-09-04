DROP DATABASE IF EXISTS `advanced`;
CREATE DATABASE IF NOT EXISTS `advanced`;
USE `advanced`;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for demo
-- ----------------------------
DROP TABLE IF EXISTS `demo`;
CREATE TABLE `demo` (
    `id` int UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '名称',
    `role_id` int UNSIGNED NOT NULL DEFAULT 0 COMMENT '角色id',
    `password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '密码(bcrypt)',
    `test1` decimal(10, 2) UNSIGNED NOT NULL DEFAULT 0.00 COMMENT '测试1',
    `test2` smallint UNSIGNED NOT NULL DEFAULT 0 COMMENT '测试2',
    `test3` year NOT NULL COMMENT '测试3',
    `test4` tinyint UNSIGNED NOT NULL DEFAULT 0 COMMENT '测试4',
    `test5` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT '' COMMENT '测试5',
    `test6` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'test6666' COMMENT '测试6',
    `test7` timestamp NULL DEFAULT NULL COMMENT '测试7',
    `test8` json NULL COMMENT '测试8',
    `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
    `test_a888` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL,
    `test_b` bigint UNSIGNED NOT NULL DEFAULT 0 COMMENT '测试b',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `a` (`name`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 1 DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci  COMMENT = 'demo';

-- ----------------------------
-- Records of demo
-- ----------------------------
-- 密码均为 123456 (bcrypt)
INSERT INTO `demo` (`id`, `name`, `role_id`, `password`, `test1`, `test2`, `test3`, `test4`, `test5`, `test6`, `test7`, `test8`, `deleted_at`, `test_a888`, `test_b`) VALUES (1, 'demo', 1, '$2a$10$KZ4bGPexK/Heh07BwEo21OzdULlvWSm0gIYUfUuh3o.1sXw8RA7ui', 0.00, 0, 0000, 0, NULL, 'test6666', NULL, NULL, NULL, NULL, 0);
INSERT INTO `demo` (`id`, `name`, `role_id`, `password`, `test1`, `test2`, `test3`, `test4`, `test5`, `test6`, `test7`, `test8`, `deleted_at`, `test_a888`, `test_b`) VALUES (2, '测试demo3', 2, '$2a$10$KZ4bGPexK/Heh07BwEo21OzdULlvWSm0gIYUfUuh3o.1sXw8RA7ui', 0.00, 0, 0000, 0, NULL, 'test6666', NULL, NULL, NULL, NULL, 0);
INSERT INTO `demo` (`id`, `name`, `role_id`, `password`, `test1`, `test2`, `test3`, `test4`, `test5`, `test6`, `test7`, `test8`, `deleted_at`, `test_a888`, `test_b`) VALUES (3, '测试6', 0, '', 0.00, 0, 0000, 0, 'aaa', 'test6666', NULL, NULL, NULL, NULL, 0);

SET FOREIGN_KEY_CHECKS = 1;
