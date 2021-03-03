DROP DATABASE money_manager;
CREATE DATABASE money_manager;
USE money_manager;

CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username  VARCHAR (32) NOT NULL,
    password VARCHAR(256) NOT NULL,
    UNIQUE (username)
);