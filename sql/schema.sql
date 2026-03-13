-- Run:
CREATE DATABASE msmareqdb CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE msmareqdb;

CREATE TABLE IF NOT EXISTS form (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `name` NVARCHAR(255) NOT NULL,
    `description` NVARCHAR(1024),
    `steps` TEXT,
    `video_link` NVARCHAR(512)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS student_rank (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `name` NVARCHAR(255) NOT NULL,
    `description` NVARCHAR(1024),
    `belt_color` NVARCHAR(32),
    `stripe_color` NVARCHAR(32),
    `stripe_count` INT,
    `requirements` TEXT,
    `form_id` INT,
    CONSTRAINT fk_rank_form FOREIGN KEY (form_id) REFERENCES form(id) ON DELETE SET NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user (
                                    `username` NVARCHAR(100) NOT NULL PRIMARY KEY,
    `first_name` NVARCHAR(255) NOT NULL,
    `last_name` NVARCHAR(255) NOT NULL,
    `password` NVARCHAR(255) NOT NULL,
    `is_admin` TINYINT(1) DEFAULT 0,
    `is_active` TINYINT(1) DEFAULT 1,
    `student_rank_id` INT,
    `allow_full_access` TINYINT(1) DEFAULT 0,
    `last_login_date` DATETIME NULL,
    `force_password_change` TINYINT(1) DEFAULT 1,
    CONSTRAINT fk_user_rank FOREIGN KEY (student_rank_id) REFERENCES student_rank(id) ON DELETE SET NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;