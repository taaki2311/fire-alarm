#!/usr/bin/env sh
set -e
cd "$(dirname "$0")"

echo "Creating temporary database file"
DATABASE=$(mktemp)
echo "Successfully created temporary database file: $DATABASE"

echo "Creating tables in database"
sqlite3 "$DATABASE" < setup.sql
echo "Successfully created tables in database"

for DIRECTORY in service website; do
    echo "Generating ORM code for $DIRECTORY"
    sea-orm-cli generate entity \
        --output-dir ./$DIRECTORY/src/database \
        --with-prelude all-allow-unused-imports \
        --database-url sqlite://"$DATABASE"?mode=ro
    echo "Successfully generated ORM code for $DIRECTORY"
done

echo "Deleting temporary database file: $DATABASE"
rm "$DATABASE"
echo "Successfully deleted temporary database file: $DATABASE"

echo "Running setup script for service"
sh ./service/setup.sh
echo "Successfully ran setup script for service"
