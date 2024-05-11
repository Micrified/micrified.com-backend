# micrified.com

Work in progress web API for my personal website.

## Configuration

A configuration file of the following format is provided as an input argument to the the server binary when starting:

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

Each service has an entry in the configuration file. At present, the `auth` and `database` services have a configurable entry. They are respectively:
1. The authentication login penalty algorithm. This will be described in a later update to this README.
2. The database login information. A set of valid credentials are required to establish a connection with the MySQL8 backend database server. 

Do note that this web API requires a particular database structure to be useable. This database structure will be described in a later update to the README. 

