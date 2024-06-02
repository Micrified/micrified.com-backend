DROP DATABASE IF EXISTS main;
CREATE DATABASE main;
USE main;

-- Page describes common fields for queried text-based content
CREATE TABLE page (
  id      MEDIUMINT NOT NULL AUTO_INCREMENT,
  created TIMESTAMP NOT NULL,
  updated TIMESTAMP NOT NULL,
  body    TEXT      NOT NULL,
  PRIMARY KEY (id)
);

-- Static describes fixed (static) pages as records with hashed names
CREATE TABLE static (
  hash    BINARY(16) NOT NULL,
  page    MEDIUMINT NOT NULL,
  PRIMARY KEY (hash),
  FOREIGN KEY (page) REFERENCES page(id)
);

-- Blog describes blog pages as unnamed auto-incrementing records
CREATE TABLE blog (
  id       MEDIUMINT    NOT NULL AUTO_INCREMENT,
  title    VARCHAR(255) NOT NULL,
  subtitle VARCHAR(255),
  page     MEDIUMINT    NOT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (page) REFERENCES page(id)
);

-- Actor is a register of auto-incrementing uniquely named users
CREATE TABLE actor (
  id   MEDIUMINT   NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL UNIQUE,
  PRIMARY KEY(id),
  INDEX(name)
);

-- Credential stores each users' hashed passphrase and accompanying salt
CREATE TABLE credential (
  id    MEDIUMINT  NOT NULL AUTO_INCREMENT,
  hash  binary(64) NOT NULL,
  salt  binary(64) NOT NULL,
  actor MEDIUMINT  NOT NULL UNIQUE,
  PRIMARY KEY (id),
  FOREIGN KEY (actor) REFERENCES actor(id)
);
