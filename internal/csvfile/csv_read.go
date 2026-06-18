package csvfile

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
)

const writeDelimiter = ','

var readDelimiters = []rune{',', ';', '\t'}

func stripUTF8BOM(data []byte) []byte {
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
}

func detectCSVDelimiter(data []byte) (rune, error) {
	var (
		bestDelimiter rune
		bestScore     int
		found         bool
	)

	for _, delimiter := range readDelimiters {
		score, ok := scoreDelimiter(data, delimiter)
		if !ok {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestDelimiter = delimiter
			found = true
		}
	}

	if !found {
		return writeDelimiter, errors.New("csv-kopfzeile unlesbar oder DupID/ID fehlen")
	}
	return bestDelimiter, nil
}

func scoreDelimiter(data []byte, delimiter rune) (int, bool) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delimiter
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return 0, false
	}

	columns := mapDupColumns(normalizeHeader(header))
	if _, ok := columns["dupid"]; !ok {
		return 0, false
	}
	if _, ok := columns["id"]; !ok {
		return 0, false
	}
	return len(header), true
}

func newCSVReader(data []byte) (*csv.Reader, error) {
	data = stripUTF8BOM(data)

	delimiter, err := detectCSVDelimiter(data)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delimiter
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	return reader, nil
}

func newCSVWriter(w io.Writer) *csv.Writer {
	writer := csv.NewWriter(w)
	writer.Comma = writeDelimiter
	return writer
}
