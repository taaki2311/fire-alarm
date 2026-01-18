/*
Csv2sql converts station information (name and any rail lines it is on) to SQL
statements. Emitted SQL statements are designed to work with the tables defined
in setup.sql. Besides the table definitations, the database should be empty
(have no records in each table) since the emitted statements start indexing the
primary keys at '1'. For an example input see wmata.csv.

# WARNING! SQL INJECTION POSSIBILITY!

DO NOT PIPE DIRECTLY TO SQL DATABASE! This is purely just a helper script whos
out should be verified before passing it to your database. It does not do any
checking for or protecting against SQL injection attacks. It is meant only to be
run once to help set up your SQL database and should not be accessable to
end-users or anyone unauthorized to have direct access to your database. Do not
come complaining to me if you get "Robert');DROP TABLE Students;--"ed. YOU HAVE
BEEN WARNED!

# Usage

	cat <input.csv> | csv2sql > <output.sql>

# Example

	Input
		,Red,Green,Blue
		Foo,0,f,false
		Bar,1,F,True
		Baz,FALSE,t,False

	Output
		BEGIN;
		INSERT INTO RailLine VALUES (1, 'Red');
		INSERT INTO RailLine VALUES (2, 'Green');
		INSERT INTO RailLine VALUES (3, 'Blue');
		INSERT INTO Station VALUES (1, 'Foo');
		INSERT INTO Station VALUES (2, 'Bar');
		INSERT INTO LineStation VALUES (2, 1);
		INSERT INTO LineStation VALUES (2, 3);
		INSERT INTO Station VALUES (3, 'Baz');
		INSERT INTO LineStation VALUES (3, 2);
		COMMIT;

# CSV Format

The header should be defined as "<don't care>,<line 1 name (string)>,<line 2
name (string)>,...". Each record should be defined as "<station name
(string)>,<is on line 1 (boolean)>,<is on line 2 (boolean)>,...".

The boolean literal must be a valid option that can be parsed by
[strconv.ParseBool]. As of this writing that is false: 0, f, F, false, False,
FALSE and true: 1, t, T, true, True, TRUE.

# Escaping dangerous character for SQL injection

Csv2sql will escape NULL and single quote for string literals inside of SQL
statements. Other characters like backslash can also be dangerous for certain
databases like MySQL or MariaDB. However this implementation is kept minimal to
standard SQL to maximize compatibility. If you are using such a database, if
will be up to you to appropriately escape the string literals in your CSV.
*/
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Creates a writer to Standard Out and writes "BEGIN;" to start a SQL
	// transaction: https://www.geeksforgeeks.org/sql/sql-transactions/
	writer := bufio.NewWriter(os.Stdout)
	_, err := fmt.Fprintln(writer, "BEGIN;")
	if nil != err {
		rollback(writer)
		forceFlush(writer)
		log.Fatal("Failed to write 'BEGIN;': ", err)
	}

	// Creates the CSV reader from Standard In and reads in the header.
	reader := csv.NewReader(os.Stdin)
	header, err := reader.Read()
	if nil != err {
		rollback(writer)
		forceFlush(writer)
		log.Fatal("Failed to read CSV header: ", err)
	}

	// Check to make sure there is at least one rail line in the network.
	// This means that the header should have at least 2 columns since
	// the first column is the station name (does not need to have a title).
	recordLen := len(header)
	if 2 > recordLen {
		rollback(writer)
		forceFlush(writer)
		log.Fatal("Network must have at least one rail line")
	}

	// Creates the records for each rail line. Starts primary id at 1
	for lineId, lineName := range header[1:] {
		// Check if line name is empty
		nameLen := len(lineName)
		if 0 >= nameLen {
			rollback(writer)
			forceFlush(writer)
			log.Fatalf("Invalid name length for line %d: %d", lineId, nameLen)
		}

		_, err = fmt.Fprintf(writer, "INSERT INTO RailLine VALUES (%d, '%s');\n", lineId+1, escapeSql(lineName))
		if nil != err {
			rollback(writer)
			forceFlush(writer)
			log.Fatal("Failed to write line insert statement: ", err)
		}
	}

	stationId := 1 // Index for the station's primary key starts at 1
	// Keep looping and reading from the CSV from Standard In until you get a EOF
	for record, err := reader.Read(); err != io.EOF; record, err = reader.Read() {
		if nil != err {
			rollback(writer)
			forceFlush(writer)
			log.Fatalf("Failed to read record for station %d: %v", stationId, err)
		}

		// Check if station name is empty
		nameLen := len(record[0])
		if 0 >= nameLen {
			rollback(writer)
			forceFlush(writer)
			log.Fatalf("Invalid name length for station %d: %d", stationId, nameLen)
		}

		// Create the record for each station.
		_, err = fmt.Fprintf(writer, "INSERT INTO Station VALUES (%d, '%s');\n", stationId, escapeSql(record[0]))
		if nil != err {
			rollback(writer)
			forceFlush(writer)
			log.Fatal("Failed to write station insert statement: ", err)
		}

		for lineId, onLine := range record[1:] { // For every line...
			isOnLine, err := strconv.ParseBool(onLine) // check if the station in on it (has 'true' in the column)
			if nil != err {
				rollback(writer)
				forceFlush(writer)
				log.Fatalf("Failed to parse boolean value for %s, line %s: %v", record[0], header[lineId+1], err)
			} else if isOnLine {
				// Create a link between the station and the line
				_, err = fmt.Fprintf(writer, "INSERT INTO LineStation VALUES (%d, %d);\n", stationId, lineId+1)
				if nil != err {
					rollback(writer)
					forceFlush(writer)
					log.Fatal("Failed to write link statement: ", err)
				}
			}
		}
		stationId++
	}
	// Terminates the SQL transaction with "COMMIT;"
	_, err = fmt.Fprintln(writer, "COMMIT;")
	if nil != err {
		rollback(writer)
		forceFlush(writer)
		log.Fatal("Failed to write 'COMMIT;': ", err)
	}
	forceFlush(writer)
}

// Called in case there was an error. Will issue a ROLLBACK to the SQL
// transaction to prevent it from executing. If there was an error doing so it
// will log it to Standard Error but otherwise continue.
func rollback(writer io.Writer) {
	_, err := fmt.Fprintln(writer, "ROLLBACK;")
	if nil != err {
		log.Println("Failed to write 'ROLLBACK;': ", err)
	}
}

// Called in case there was an error. Will forcibly flush the writer
// (particularly the "ROLLBACK;") will to written to Standard Out. If there was
// an error doing so it will log it to Standard Error but otherwise continue.
func forceFlush(writer *bufio.Writer) {
	err := writer.Flush()
	if nil != err {
		log.Println("Failed to flush the writer: ", err)
	}
}

// Escape specific characters from the statement before passing it to the SQL
// string. Does so by preforming a text replacement using [stringsReplacement].
// It is an array of pairs of strings, where the first string in the pair is the
// replacee and the second is the replacer. Currently only "NULL -> <empty>" and
// "<single quote> -> <single quote><single quote>" are the only pairs but more
// can be added by appending them to the end of [stringReplacements].
func escapeSql(statement string) string {
	stringReplacements := [][2]string{{"\x00", ""}, {"'", "''"}} // <- Add if needed here
	for _, stringReplacement := range stringReplacements {
		statement = strings.ReplaceAll(statement, stringReplacement[0], stringReplacement[1])
	}
	return statement
}
