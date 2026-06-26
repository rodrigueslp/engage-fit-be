package services

import (
	"context"
	"io"
)

type ReportExporter interface {
	WriteCSV(ctx context.Context, writer io.Writer, headers []string, rows [][]string) error
}
