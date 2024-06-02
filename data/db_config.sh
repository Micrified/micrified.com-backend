#!/bin/sh 
# db_config: Generate JSON configuration file as a server argument
USAGE="$0 <unixsocket> <username> <password> <output-file>"

# Argument check
if [ "$#" -ne "4" ]; then 
  echo $USAGE 1>&2; exit 1;
fi

cat << EOF
{
  "Auth" : {
    "Base"   : 2,
    "Factor" : 2,
    "Limit"  : 8,
    "Retry"  : 3
  },
  "Database" : {
    "UnixSocket" : "$1",
    "Username"   : "$2",
    "Password"   : "$3",
    "Database"   : "main"
  },
  "Host" : "localhost",
  "Port" : "3070"
}
EOF > $4

exit 0
