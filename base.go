package mailing

import (
	"context"
)

type IMailing interface {
	SendSimple(ctx context.Context, toAddresses []string, subject, body, textType string) error
	SendWithFile(ctx context.Context, toAddresses []string, subject, filePath string) error
}
