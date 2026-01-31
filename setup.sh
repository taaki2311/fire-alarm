#!/bin/sh
set -e

DATABASE=$(mktemp)
echo "Created temporary file: $DATABASE"

sqlite3 "$DATABASE" < setup.sql
echo "Creating tables in database"

for DIRECTORY in service website; do
    sea-orm-cli generate entity \
        --output-dir ./$DIRECTORY/src/database \
        --with-prelude all-allow-unused-imports \
        --database-url sqlite://"$DATABASE"?mode=ro
done

rm "$DATABASE"
echo "Deleting temporary file"

TIMESTAMP=./service/timestamp.txt
printf "1985-04-12T23:20:50.52Z" > $TIMESTAMP
echo "Writing dummy timestamp to $TIMESTAMP"
