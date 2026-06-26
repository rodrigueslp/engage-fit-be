package parsers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

type CheckinParser struct{}

func NewCheckinParser() CheckinParser {
	return CheckinParser{}
}

func (p CheckinParser) Parse(ctx context.Context, reader io.Reader, source domain.Source, filename string) ([]services.ParsedCheckin, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return parseCSV(reader, source)
	case ".xlsx":
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		return parseXLSX(data, source)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func parseCSV(reader io.Reader, source domain.Source) ([]services.ParsedCheckin, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	delimiter := ','
	firstLine := strings.SplitN(string(data), "\n", 2)[0]
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		delimiter = ';'
	}

	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true

	rows, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	return parseRows(rows, source)
}

func parseXLSX(data []byte, source domain.Source) ([]services.ParsedCheckin, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	sharedStrings, err := readSharedStrings(zipReader)
	if err != nil {
		return nil, err
	}

	sheet, err := readFirstSheet(zipReader, sharedStrings)
	if err != nil {
		return nil, err
	}

	return parseRows(sheet, source)
}

func parseRows(rows [][]string, source domain.Source) ([]services.ParsedCheckin, error) {
	if len(rows) < 2 {
		return []services.ParsedCheckin{}, nil
	}

	headerIndex, headers := findHeaderRow(rows)
	if headerIndex < 0 {
		return []services.ParsedCheckin{}, nil
	}

	result := make([]services.ParsedCheckin, 0, len(rows)-headerIndex-1)

	for _, row := range rows[headerIndex+1:] {
		if emptyRow(row) {
			continue
		}

		name := value(row, headers, "name")
		dateValue := value(row, headers, "date")
		if name == "" || dateValue == "" {
			continue
		}

		checkinDate, err := parseDate(dateValue)
		if err != nil {
			return nil, err
		}

		var checkinTime *time.Time
		if timeValue := value(row, headers, "time"); timeValue != "" {
			parsedTime, err := parseTime(timeValue)
			if err != nil {
				return nil, err
			}
			checkinTime = &parsedTime
		} else if parsedTime, ok := parseTimeFromDateTime(dateValue); ok {
			checkinTime = &parsedTime
		}

		result = append(result, services.ParsedCheckin{
			StudentName:       name,
			StudentEmail:      value(row, headers, "email"),
			StudentPhone:      value(row, headers, "phone"),
			StudentExternalID: value(row, headers, "external_id"),
			CheckinDate:       checkinDate,
			CheckinTime:       checkinTime,
			Source:            source,
		})
	}

	return result, nil
}

func findHeaderRow(rows [][]string) (int, map[string]int) {
	for index, row := range rows {
		headers := mapHeaders(row)
		if _, hasName := headers["name"]; !hasName {
			continue
		}
		if _, hasDate := headers["date"]; !hasDate {
			continue
		}
		return index, headers
	}
	return -1, map[string]int{}
}

func mapHeaders(headers []string) map[string]int {
	result := map[string]int{}
	for index, header := range headers {
		normalized := normalizeHeader(header)
		switch normalized {
		case "id", "externalid", "external_id", "codigo", "matricula", "studentid", "userid", "iddowellhub":
			result["external_id"] = index
		case "name", "nome", "aluno", "student", "cliente", "usuario", "user", "colaborador", "visitante":
			result["name"] = index
		case "email", "e_mail":
			result["email"] = index
		case "phone", "telefone", "celular", "whatsapp":
			result["phone"] = index
		case "date", "data", "checkindate", "data_checkin", "checkin_date", "validadoem":
			result["date"] = index
		case "time", "hora", "checkintime", "hora_checkin", "checkin_time":
			result["time"] = index
		}
	}
	return result
}

func normalizeHeader(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(" ", "", "-", "_", ".", "", "/", "_")
	return replacer.Replace(value)
}

func value(row []string, headers map[string]int, key string) string {
	index, ok := headers[key]
	if !ok || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func emptyRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func parseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	formats := []string{"2006-01-02", "02/01/2006", "01/02/2006", "2006/01/02", "02-01-2006", "2006-01-02 15:04", "2006-01-02 15:04:05", "02/01/2006 15:04", "02/01/2006 15:04:05", time.RFC3339}
	for _, format := range formats {
		parsed, err := time.Parse(format, value)
		if err == nil {
			return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}

	if serial, err := strconv.ParseFloat(value, 64); err == nil {
		return excelSerialDate(serial), nil
	}

	return time.Time{}, fmt.Errorf("invalid checkin date: %s", value)
}

func parseTimeFromDateTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	formats := []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "02/01/2006 15:04", "02/01/2006 15:04:05", time.RFC3339}
	for _, format := range formats {
		parsed, err := time.Parse(format, value)
		if err == nil {
			return time.Date(1970, 1, 1, parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC), true
		}
	}
	return time.Time{}, false
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	formats := []string{"15:04:05", "15:04"}
	for _, format := range formats {
		parsed, err := time.Parse(format, value)
		if err == nil {
			return time.Date(1970, 1, 1, parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC), nil
		}
	}

	if serial, err := strconv.ParseFloat(value, 64); err == nil {
		seconds := int(serial * 24 * 60 * 60)
		return time.Date(1970, 1, 1, seconds/3600, (seconds%3600)/60, seconds%60, 0, time.UTC), nil
	}

	return time.Time{}, fmt.Errorf("invalid checkin time: %s", value)
}

func excelSerialDate(serial float64) time.Time {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	return base.Add(time.Duration(serial*24) * time.Hour)
}

type sharedStringTable struct {
	Items []sharedStringItem `xml:"si"`
}

type sharedStringItem struct {
	Text string `xml:"t"`
}

func readSharedStrings(zipReader *zip.Reader) ([]string, error) {
	file := findZipFile(zipReader, "xl/sharedStrings.xml")
	if file == nil {
		return []string{}, nil
	}

	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var table sharedStringTable
	if err := xml.NewDecoder(reader).Decode(&table); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(table.Items))
	for _, item := range table.Items {
		result = append(result, item.Text)
	}
	return result, nil
}

type worksheet struct {
	Rows []worksheetRow `xml:"sheetData>row"`
}

type worksheetRow struct {
	Cells []worksheetCell `xml:"c"`
}

type worksheetCell struct {
	Ref   string `xml:"r,attr"`
	Type  string `xml:"t,attr"`
	Value string `xml:"v"`
	Text  string `xml:"is>t"`
}

func readFirstSheet(zipReader *zip.Reader, sharedStrings []string) ([][]string, error) {
	file := findZipFile(zipReader, "xl/worksheets/sheet1.xml")
	if file == nil {
		return nil, errors.New("sheet1.xml not found")
	}

	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var sheet worksheet
	if err := xml.NewDecoder(reader).Decode(&sheet); err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(sheet.Rows))
	for _, sheetRow := range sheet.Rows {
		row := []string{}
		for _, cell := range sheetRow.Cells {
			column := cellColumnIndex(cell.Ref)
			for len(row) <= column {
				row = append(row, "")
			}
			row[column] = cellValue(cell, sharedStrings)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func findZipFile(zipReader *zip.Reader, name string) *zip.File {
	for _, file := range zipReader.File {
		if file.Name == name {
			return file
		}
	}
	return nil
}

func cellValue(cell worksheetCell, sharedStrings []string) string {
	switch cell.Type {
	case "s":
		index, err := strconv.Atoi(cell.Value)
		if err == nil && index >= 0 && index < len(sharedStrings) {
			return sharedStrings[index]
		}
		return ""
	case "inlineStr":
		return cell.Text
	default:
		return cell.Value
	}
}

func cellColumnIndex(ref string) int {
	index := 0
	for _, char := range ref {
		if char < 'A' || char > 'Z' {
			break
		}
		index = index*26 + int(char-'A'+1)
	}
	if index == 0 {
		return 0
	}
	return index - 1
}
