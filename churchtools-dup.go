package main

/*
 * churchtools-dup: A tool to detect and handle duplicates in ChurchTools instances.
 *
 * churchtools-dup is a command-line tool for identifying and exporting potential duplicate persons from ChurchTools instances.
 * It analyzes person data, searches for possible duplicates based on names, addresses, and other criteria,
 * and exports found duplicates as a clear CSV file for further review or cleanup.
 * The tool supports selecting a campus (location), various filtering options, and can be used interactively.
 * Its primary purpose is to help organizations such as churches maintain clean data and efficiently resolve duplicate entries in ChurchTools.
 *
 *
 * Version: 1.1.0.5 (in version.go zu ändern)
 *
 * Author: Jan Neuhaus, VAYA Consulting, https://vaya-consulting.de/development
 *
 * Repository: https://github.com/janmz/ChurchToolsDup
 *
 * ChangeLog:
 *  17.06.26	1.0.0	Initial version (forked from churchtools-invite)
 *
 * (c)2026 Jan Neuhaus, VAYA Consulting
 *
 */

import (
	"fmt"
	"os"

	cmd "github.com/janmz/churchtools-dup/cmd"
)

func main() {
	if err := cmd.Execute(fmt.Sprintf("%s (%s)", Version, BuildTime)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
