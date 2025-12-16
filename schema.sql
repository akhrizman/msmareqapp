DROP database msmareqdb;

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

-- Sample forms and ranks
INSERT INTO form (name, description, steps, video_link)
VALUES ('None', 'No Form Required', 'N/A', ''),
       ('Gicho Hyung Il Bu', 'Beginner Form Number 1', 'Step 1\nStep 2', ''),
       ('Gicho Hyung Yi Bu', 'Beginner Form Number 2', 'Step 1\nStep 2', ''),
       ('Gicho Hyung Sahm Bu', 'Beginner Form Number 3', 'Step 1\nStep 2', ''),
       ('Pyang An Cho Dan', 'Peaceful and Calm Confidence 1st Level', 'Step 1\nStep 2', ''),
       ('Pyang An Yi Dan', 'Peaceful and Calm 2nd Level', 'Step 1\nStep 2', ''),
       ('Pyang An Sahm Dan', 'Peaceful and Calm 3rd Level', 'Step 1\nStep 2', ''),
       ('Pyang An Sa Dan', 'Peaceful and Calm 4th Level', 'Step 1\nStep 2', ''),
       ('Pyang An Oh Dan', 'Peaceful and Calm 5th Level', 'Step 1\nStep 2', ''),
       ('Bassai', 'To Penetrate a Fortress', 'Step 1\nStep 2', ''),
       ('Ro Hai', 'Vision of a Crane', 'Step 1\nStep 2', ''),
       ('Nianchi Cho Dan', 'Warrior on a[n] [Iron] Horse', 'Step 1\nStep 2', ''),
       ('Koon San Goon', 'Temple of Mercy', 'Step 1\nStep 2', '');

INSERT INTO student_rank (id, name, description, requirements, form_id)
VALUES (1, 'New Student', 'White Belt', 'No Requirements', 1),
       (2, '12th Gup', 'White Belt (yellow stripe)', 'Requirements for 12th Gup', 1),
       (3, '11th Gup', 'Senior White Belt (2 yellow stripes)', 'Requirements for 11th Gup', 1),
       (4, '10th Gup', 'Yellow Belt', 'Requirements for 10th Gup', 2),
       (5, '9th Gup', 'Senior Yellow Belt (blue stripe)', 'Requirements for 9th Gup', 2),
       (6, '8th Gup', 'Blue Belt', 'Requirements for 8th Gup', 3),
       (7, '7th Gup', 'Senior Blue Belt (green stripe)', 'Requirements for 7th Gup', 4),
       (8, '6th Gup', 'Green Belt', 'Requirements for 6th Gup', 5),
       (9, '5th Gup', 'Senior Green Belt (purple stripe)', 'Requirements for 5th Gup', 6),
       (10, '4th Gup', 'Purple Belt', 'Requirements for 4th Gup', 7),
       (11, '3rd Gup', 'Senior Purple Belt (brown stripe)', 'Requirements for 3rd Gup', 8),
       (12, '2nd Gup', 'Brown Belt', 'Requirements for 2nd Gup', 9),
       (13, '1st Gup', 'Senior Brown Belt (black stripe)', 'Requirements for 1st Gup', 10),
       (14, '1st Dan', '1st Degree Black Belt', 'Requirements for 1st Dan', 11),
       (15, '2nd Dan', '2nd Degree Black Belt', 'Requirements for 2nd Dan', 12),
       (16, '3rd Dan', '3rd Degree Black Belt', 'Requirements for 3rd Dan', 13),
       (17, '4th Dan', '4th Degree Black Belt', 'No Testing Requirements', 1),
       (18, '5th Dan', '5th Degree Black Belt', 'No Testing Requirements', 1),
       (19, '6th Dan', '6th Degree Black Belt', 'No Testing Requirements', 1),
       (20, '7th Dan', '7th Degree Black Belt', 'No Testing Requirements', 1),
       (21, '8th Dan', '8th Degree Black Belt', 'No Testing Requirements', 1),
       (22, '9th Dan', '9th Degree Black Belt', 'No Testing Requirements', 1),
       (23, 'Grand master', 'Highest Achievable Rank', 'No Testing Requirements', 1);
