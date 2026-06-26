package reports

import (
	"context"
	"encoding/csv"
	"io"
)

type CSVExporter struct{}

func NewCSVExporter() CSVExporter {
	return CSVExporter{}
}

func (e CSVExporter) WriteCSV(ctx context.Context, writer io.Writer, headers []string, rows [][]string) error {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(headers); err != nil {
		return err
	}
	if err := csvWriter.WriteAll(rows); err != nil {
		return err
	}
	csvWriter.Flush()
	return csvWriter.Error()
}
