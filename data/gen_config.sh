#!/bin/sh 
# gen_config: Generate JSON configuration file as a server argument
USAGE="$0 <unixsocket> <username> <password>"

# Argument check
if [ "$#" -ne "3" ]; then 
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
EOF

exit 0
