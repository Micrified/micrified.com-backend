# micrified.com / backend

Backend API for a personal website. Always a work in progress!

## Build

This project is written with Go version `1.22` in mind. Note that previous versions of Go may not be compatible! The code relies on builtin parametric polymorphism (like the `min` and `max` functions) introduced in Go 1.21 (see the [release notes](https://tip.golang.org/doc/go1.21)).

To build the repository, simply execute:
```
go build
```
This outputs a server binary aptly named: `server`

---

## Run

Running the server requires two preconditions be met:
1. A valid configuration file is present (to be fed in as a program argument)
2. A MySQL server is running (described below, with matching credentials to the configuration)

Provided these are satisfied, the server may simply be run with:
```
./server <my-config-file>.json
```

### Configuration file
Provide a configuration file with the following format:
```json
{
  "Auth" : {
    "Base"   : 2,
    "Factor" : 2,
    "Limit"  : 8,
    "Retry"  : 3
  },
  "Database" : {
    "UnixSocket" : "/opt/local/var/run/mysql8/mysqld.sock",
    "Username"   : "my-username",
    "Password"   : "my-password",
    "Database"   : "my-database-name"
  },
  "Host" : "localhost",
  "Port" : "3070"
}
```
Aside from the server host and port, each *service* (implemented within the `services` directory) embeds a service configuration within the main configuration struct defined in `config.go`. This exposes behavior that should be defined by the user. There are presently two services:
1. The authentication login penalty algorithm. This will be described in a later update to this README.
2. The database login information. A set of valid credentials are required to establish a connection with the MySQL8 backend database server. 

### MySQL Server

As mentioned earlier, MySQL is required for operating the server. The server connects to the databse upon starting using the configuration information provided as input. Installation of MySQL 8.0 and higher is recommended. A user must also be added to the database after installation and the login credentials supplied in the `Username` and `Password` fields of the configuration file. This user must have the following privileges in order to function:`INSERT`, `UPDATE`, `DELETE`, `SELECT`, and `REFERENCES`. See `data/db_setup.sql` for an example of how these permissions can be granted.

Finally, a database table schema and some initialization data is required in order for testing to function. Perform the following in order to complete the configuration:
1. Create the database and table schema: `mysql -uroot -p<ROOT-PASSWORD> < data/db_schema.sql`
2. Install test user and initial login data with: `mysql -uroot -p<ROOT-PASSWORD> < data/db_setup.sql`

---

## Test

To test the API, a successful build is required. All preconditions needed for running the server must also be satisified. In summary:
1. A configured MySQL database must be running
2. `server` must be running with a valid configuration

Before executing the test, environment variables containing the server credentials must be set. Do note that these credentials are not the same as those for the MySQL database. Rather, these are independent and are used to test user login functionality where the user is a record installed within the database using the setup scripts. In summary, set the following (assuming `zsh`):

```zsh
export TEST_HOSTNAME="http://<host>:<port>"
export TEST_USERNAME="<TEST-USERNAME>"
export TEST_PASSPHRASE="<TEST-PASSWORD>"
```

Finally, you may run the tests by executing: `go test <filename>` for each file within the `test` subdirectory:
```
go test test/auth_test.go 
go test test/blog_test.go 
go test test/static_test.go 
...
```
