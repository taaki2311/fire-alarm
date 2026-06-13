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
		Line,Red,Green,Blue
		Ruby,255,0,0
		Emerald,0,255,0
		Sapphire,0,0,255

	Input (stations.csv)
		Station,Ruby,Emerald,Sapphire
		Foo,1,0,f
		Bar,t,T,F
		Baz,false,true,True

	Output
		BEGIN;
		INSERT INTO RailLines VALUES (1, 'Ruby', 255, 0, 0);
		INSERT INTO RailLines VALUES (2, 'Emerald', 0, 255, 0);
		INSERT INTO RailLines VALUES (3, 'Sapphire', 0, 0, 255);
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

The "lines" table should list all of the lines in the train network followed
by the red, green, and blue value for their associated color. The "stations"
table should include all the lines as columns in the header and list out all
of the stations in the network followed by whether they are on the line using
a boolean. The first columns for each table is assumed to be the name of the
line or station, the actual value in the header for the first column is
ignored. See wmata/ for an actual example.

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
	stationsPath := flag.String("stations", "stations.csv", "CSV file for the stations")
	flag.Parse()

	writer := bufio.NewWriter(os.Stdout)
	defer func(writer *bufio.Writer) {
		if err := writer.Flush(); nil != err {
			log.Fatalln("Failed to flush writer:", err)
		}
	}(writer)

	if err := execFromCsvFile(*linesPath, lineStatements, writer); nil != err {
		log.Fatalln("Failed to generate rail line SQL statements:", err)
	}

	if err := execFromCsvFile(*stationsPath, stationStatements, writer); nil != err {
		log.Fatalln("Failed to generate station SQL statements:", err)
	}
}

// Generate the SQL statements for populating the 'RailLines' table.
func lineStatements(reader *csv.Reader, writer io.Writer) error {
	reader.FieldsPerRecord = 4 // Line Name, Red, Green, and Blue
	header, err := reader.Read()
	if nil != err {
		return fmt.Errorf("Failed to read CSV header: %w", err)
	}

	redIndex, greenIndex, blueIndex := 0, 0, 0
	for i, entry := range header[1:] {
		entry = strings.TrimSpace(entry)
		if strings.EqualFold(entry, "Red") {
			redIndex = i + 1
		} else if strings.EqualFold(entry, "Green") {
			greenIndex = i + 1
		} else if strings.EqualFold(entry, "Blue") {
			blueIndex = i + 1
		}
	}

	if 0 == redIndex {
		return fmt.Errorf("Failed to find Red column")
	}
	if 0 == greenIndex {
		return fmt.Errorf("Failed to find Green column")
	}
	if 0 == blueIndex {
		return fmt.Errorf("Failed to find Blue column")
	}

	lineId := 1
	for record, err := reader.Read(); io.EOF != err; record, err = reader.Read() {
		if nil != err {
			return fmt.Errorf("Failed to read record for line %d: %w", lineId, err)
		}

		lineName := strings.TrimSpace(record[0])
		if nameLen := len(lineName); 0 >= nameLen {
			return fmt.Errorf("Invalid name length for line %d: %d", lineId, nameLen)
		}

		red, err := parseUint8(record[redIndex])
		if nil != err {
			return fmt.Errorf("Failed to parse red value for %s: %w", lineName, err)
		}
		green, err := parseUint8(record[greenIndex])
		if nil != err {
			return fmt.Errorf("Failed to parse green value for %s: %w", lineName, err)
		}
		blue, err := parseUint8(record[blueIndex])
		if nil != err {
			return fmt.Errorf("Failed to parse blue value for %s: %w", lineName, err)
		}

		if _, err = fmt.Fprintf(writer, "INSERT INTO RailLines VALUES (%d, '%s', %d, %d, %d);\n",
			lineId, escapeSqlString(lineName), red, green, blue); nil != err {
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

	// The header should have at least 2 fields since the first column is the station name.
	headerNumFields := len(header)
	if 2 > headerNumFields {
		return fmt.Errorf("Network must have at least one rail line")
	}
	reader.FieldsPerRecord = headerNumFields

	stationId := 1
	for record, err := reader.Read(); io.EOF != err; record, err = reader.Read() {
		if nil != err {
			return fmt.Errorf("Failed to read record for station %d: %w", stationId, err)
		}

		stationName := strings.TrimSpace(record[0])
		if nameLen := len(stationName); 0 >= nameLen {
			return fmt.Errorf("Invalid name length for station %d: %d", stationId, nameLen)
		}

		if _, err = fmt.Fprintf(writer, "INSERT INTO Stations VALUES (%d, '%s');\n", stationId, escapeSqlString(stationName)); nil != err {
			return fmt.Errorf("Failed to write station insert statement: %w", err)
		}

		for lineId, onLine := range record[1:] {
			if isOnLine, err := strconv.ParseBool(strings.TrimSpace(onLine)); nil != err {
				return fmt.Errorf("Failed to parse boolean value for %s, line %s: %w", stationName, header[lineId+1], err)
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

// Escape specific characters from the statement before passing it to the SQL string.
// Currently only "NUL -> <empty>" and "<single quote> -> <single quote><single quote>"
// are the only pairs but more can be added by appending them to the argument for
// [strings.NewReplacer].
func escapeSqlString(statement string) string {
	return strings.NewReplacer("\x00", "", "'", "''").Replace(statement)
}

// Converts decimal string of a number to a unsigned byte.
func parseUint8(s string) (uint8, error) {
	number, err := strconv.ParseUint(strings.TrimSpace(s), 10, 8)
	return uint8(number), err
}

// Manages file operations for [performTransaction].
func execFromCsvFile(path string, statements csv2sqlStatements, writer io.Writer) error {
	file, err := os.Open(path)
	if nil != err {
		return fmt.Errorf("Failed to open %s: %w", path, err)
	}
	defer func(file *os.File, path string) {
		if err := file.Close(); nil != err {
			log.Printf("Failed to close %s: %v\n", path, err)
		}
	}(file, path)

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	return performTransaction(statements, reader, writer)
}

// Function prototype for generating SQL statements from a CSV.
type csv2sqlStatements func(*csv.Reader, io.Writer) error

// Wraps the SQL statements generator functions in a SQL transaction and returns any error from them.
func performTransaction(statements csv2sqlStatements, reader *csv.Reader, writer io.Writer) error {
	if _, err := fmt.Fprintln(writer, "BEGIN;"); nil != err {
		return fmt.Errorf("Failed to begin SQL transaction: %w", err)
	}

	err := statements(reader, writer)
	conclusion := "ROLLBACK"
	if nil == err {
		conclusion = "COMMIT"
	}

	if _, err := fmt.Fprintf(writer, "%s;\n", conclusion); nil != err {
		return fmt.Errorf("Failed to end SQL transaction: %w", err)
	}
	return err
}
