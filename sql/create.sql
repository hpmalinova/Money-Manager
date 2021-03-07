DROP DATABASE money_manager;
CREATE DATABASE money_manager;
USE money_manager;

-- todo add wallet
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username  VARCHAR (32) NOT NULL,
    password VARCHAR(256) NOT NULL,
    UNIQUE (username)
);

-- The action_user_id represent the id of the user who has performed the most recent status field update.
-- user_one_id is smaller than user_two_id
CREATE TABLE friendship (
    user_one_id INT NOT NULL,
    user_two_id INT NOT NULL,
    status enum('pending','accepted','declined') NOT NULL,
    action_user_id INT NOT NULL,
    UNIQUE (user_one_id, user_two_id)
);

CREATE TABLE groups (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name  VARCHAR (32) NOT NULL,
    participant_id INT NOT NULL,
    UNIQUE (name, participant_id)
);

CREATE TABLE categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    c_type enum('expense','income') NOT NULL,
    name  VARCHAR (32) NOT NULL,
    UNIQUE(name)
);

-- todo foreign key? - uid/category_id
-- todo add Date
CREATE TABLE money_history (
    uid INT NOT NULL,
    amount INT NOT NULL,
    category_id INT NOT NULL,
    description  VARCHAR (128)
);

CREATE TABLE debt_status (
    id INT AUTO_INCREMENT PRIMARY KEY,
    status enum('ongoing','pending') NOT NULL,
    amount INT NOT NULL
);

-- todo foreign key? - creditor/debtor/category_id/status_id
-- todo delete status when debt is payed
CREATE TABLE debts (
    creditor INT NOT NULL,
    debtor INT NOT NULL,
    amount INT NOT NULL,
    category_id INT NOT NULL,
    description  VARCHAR (128),
    status_id INT NOT NULL
);
