/*
Csv2sql converts station information (name and any rail lines it is on) to SQL
statements. Emitted SQL statements are designed to work with the tables defined
in setup.sql. The database should be empty (have no records in each table)
since the emitted statements start indexing the primary keys at '1'.

# WARNING! SQL INJECTION POSSIBILITY!

DO NOT PIPE DIRECTLY TO SQL DATABASE! This is purely just a helper script whose
output should be verified before passing it to your database. It does not do any
checking for or protecting against SQL injection attacks. It is meant only to be
run once to help set up your SQL database and should not be accessible to
end-users or anyone unauthorized to have direct access to your database. Do not
come complaining to me if you get "Robert');DROP TABLE Students;--"ed. YOU HAVE
BEEN WARNED!

# Usage

	csv2sql -lines lines.csv -stations stations.csv > output.sql

# Example

	Input (lines.csv)
		Red,255,0,0
		Green,0,255,0
		Blue,0,0,255

	Input (stations.csv)
		Foo,true,false,false
		Bar,true,true,false
		Baz,false,true,true

	Output
		BEGIN;
		INSERT INTO RailLines VALUES (1, 'Red', 255, 0, 0);
		INSERT INTO RailLines VALUES (2, 'Green', 0, 255, 0);
		INSERT INTO RailLines VALUES (3, 'Blue', 0, 0, 255);
		COMMIT;
		BEGIN;
		INSERT INTO Stations VALUES (1, 'Foo');
		INSERT INTO LineStations VALUES (1, 1);
		INSERT INTO Stations VALUES (2, 'Bar');
		INSERT INTO LineStations VALUES (1, 2);
		INSERT INTO LineStations VALUES (2, 2);
		INSERT INTO Stations VALUES (3, 'Baz');
		INSERT INTO LineStations VALUES (2, 3);
		INSERT INTO LineStations VALUES (3, 3);
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
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// Reads rail line and station data from CSV files specified via command-line flags
// and generates SQL INSERT statements. Output is written to Standard Out.
func main() {
	linesPath := flag.String("lines", "lines.csv", "CSV file for the rail lines")
	stationsPath := flag.String("stations", "stations.csv", "CVS file for the stations")
	flag.Parse()

	writer := bufio.NewWriter(os.Stdout)
	defer func(writer *bufio.Writer) {
		if err := writer.Flush(); nil != err {
			log.Panicln("Failed to flush writer:", err)
		}
	}(writer)

	if err := execFromCsvFile(*linesPath, lineStatements, writer); nil != err {
		log.Panicln("Failed to generate rail line SQL statements:", err)
	}

	if err := execFromCsvFile(*stationsPath, stationStatements, writer); nil != err {
		log.Panicln("Failed to generate station SQL statements:", err)
	}
}

// Generate the SQL statements for populating the 'RailLines' table.
func lineStatements(reader *csv.Reader, writer io.Writer) error {
	header, err := reader.Read()
	if nil != err {
		return fmt.Errorf("Failed to read CSV header: %w", err)
	}

	if recordLen := len(header); 4 > recordLen {
		return fmt.Errorf(
			"Invalid number of columns: Expected at least 4 (Station Name, Red, Green, and Blue), Got %d", recordLen)
	}

	lineId := 1
	for record, err := reader.Read(); err != io.EOF; record, err = reader.Read() {
		if nil != err {
			return fmt.Errorf("Failed to read record for line %d: %w", lineId, err)
		}

		lineName := record[0]
		red, err := parseUint8(record[1])
		if nil != err {
			return fmt.Errorf("Failed to parse red value for %s: %w", lineName, err)
		}
		green, err := parseUint8(record[2])
		if nil != err {
			return fmt.Errorf("Failed to parse green value for %s: %w", lineName, err)
		}
		blue, err := parseUint8(record[3])
		if nil != err {
			return fmt.Errorf("Failed to parse blue value for %s: %w", lineName, err)
		}

		if _, err = fmt.Fprintf(writer, "INSERT INTO RailLines VALUES (%d, '%s', %d, %d, %d);\n",
			lineId, escapeSql(lineName), red, green, blue); nil != err {
			return fmt.Errorf("Failed to write line insert statement: %w", err)
		}
		lineId++
	}
	return nil
}

// Generate the SQL statements for populating the 'Stations' table.
func stationStatements(reader *csv.Reader, writer io.Writer) error {
	header, err := reader.Read()
	if nil != err {
		return fmt.Errorf("Failed to read CSV header: %w", err)
	}

	// The header should have at least 2 columns since the first column is the station name (does not need to have a title).
	if recordLen := len(header); 2 > recordLen {
		return fmt.Errorf("Network must have at least one rail line")
	}

	stationId := 1
	for record, err := reader.Read(); err != io.EOF; record, err = reader.Read() {
		if nil != err {
			return fmt.Errorf("Failed to read record for station %d: %w", stationId, err)
		}

		if nameLen := len(record[0]); 0 >= nameLen {
			return fmt.Errorf("Invalid name length for station %d: %d", stationId, nameLen)
		}

		if _, err = fmt.Fprintf(writer, "INSERT INTO Stations VALUES (%d, '%s');\n", stationId, escapeSql(record[0])); nil != err {
			return fmt.Errorf("Failed to write station insert statement: %w", err)
		}

		for lineId, onLine := range record[1:] {
			if isOnLine, err := strconv.ParseBool(onLine); nil != err {
				return fmt.Errorf("Failed to parse boolean value for %s, line %s: %w", record[0], header[lineId+1], err)
			} else if isOnLine {
				if _, err = fmt.Fprintf(writer, "INSERT INTO LineStations VALUES (%d, %d);\n", lineId+1, stationId); nil != err {
					return fmt.Errorf("Failed to write link statement: %w", err)
				}
			}
		}
		stationId++
	}
	return nil
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

// Converts decimal string of a number to a unsigned byte.
func parseUint8(s string) (uint8, error) {
	number, err := strconv.ParseUint(s, 10, 8)
	return uint8(number), err
}

// Function prototype for generating SQL statements from a CSV.
type csv2sqlStatements func(*csv.Reader, io.Writer) error

// Wraps the SQL statements generator functions in a SQL transaction and logs any error from them.
func preformTransaction(statements csv2sqlStatements, reader *csv.Reader, writer io.Writer) error {
	if _, err := fmt.Fprintln(writer, "BEGIN;"); nil != err {
		return fmt.Errorf("Failed to begin SQL transaction: %w", err)
	}

	err := statements(reader, writer)
	if nil == err {
		_, err = fmt.Fprintln(writer, "COMMIT;")
	} else {
		log.Println(err)
		_, err = fmt.Fprintln(writer, "ROLLBACK;")
	}

	if nil != err {
		return fmt.Errorf("Failed to end SQL transaction: %w", err)
	}
	return nil
}

// Manages file operations for [preformTransaction].
func execFromCsvFile(path string, statements csv2sqlStatements, writer io.Writer) error {
	file, err := os.Open(path)
	if nil != err {
		return fmt.Errorf("Failed to open %s: %w", path, err)
	}
	if err := preformTransaction(statements, csv.NewReader(file), writer); nil != err {
		return fmt.Errorf("Failed to wrap SQL statements in a transaction: %w", err)
	}
	if err := file.Close(); nil != err {
		return fmt.Errorf("Failed to close %s: %w", path, err)
	}
	return nil
}
