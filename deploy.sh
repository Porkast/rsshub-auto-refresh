#!/bin/bash
chmod 755 rsshub-refresh

echo "start running..."

# shellcheck disable=SC2046
# shellcheck disable=SC2006
kill -9 `cat pidfile.txt`

rm pidfile.txt

nohup ./rsshub-refresh & echo $! > pidfile.txt

echo "end"

exit
