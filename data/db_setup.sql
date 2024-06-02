
-- Create a MySQL user for the server with database privileges
DROP USER IF EXISTS "tester"@"localhost";
CREATE USER "tester"@"localhost" IDENTIFIED BY "password";
GRANT INSERT, UPDATE, DELETE, SELECT, REFERENCES ON *.* TO "tester"@"localhost";
FLUSH PRIVILEGES;

-- Install a user-account into the database for the server to use. The hash
-- and salt are predefined. The user-account credentials are:
-- user:       tester
-- passphrase: password
USE main;
DELETE a, b FROM credential AS a INNER JOIN actor AS b ON a.actor = b.id;
INSERT INTO actor (name) VALUES ("tester");
INSERT INTO credential (hash,salt,actor) VALUES (
  UNHEX("a5ee08f8e3abe7d592f6de77f1d3298a1149eba68b97f091c90b7736a1be63ab2d425f94c5346cac64807f20f654c5ad9063a4d12902c5e45533491215754883"),
  UNHEX("5dde05112957f4d0fbbd944a45dca6287150e00a474db99592c7bc0ac3db5272e3adf8acbfb198a1e6998b943dc867cc92e4bb219743fdd3f07df97f54610647"),
  LAST_INSERT_ID());
