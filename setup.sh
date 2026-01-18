#!/bin/sh
set -e

DATABASE=temp.sqlite

sqlite3 $DATABASE < setup.sql
sea-orm-cli generate entity --output-dir ./service/src/database --with-prelude all-allow-unused-imports --database-url sqlite://$DATABASE?mode=ro
sea-orm-cli generate entity --output-dir ./website/src/database --with-prelude all-allow-unused-imports --database-url sqlite://$DATABASE?mode=ro
rm $DATABASE
printf "1985-04-12T23:20:50.52Z" > ./service/timestamp.txt
