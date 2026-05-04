package inventory

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/shopspring/decimal"
)

type ImportResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// ImportItemsCSV reads a CSV file and bulk-creates items.
// Expected columns: sku, name, unit, category, description, min_stock_qty
// Rows with duplicate SKUs are skipped, not failed.
func ImportItemsCSV(ctx context.Context, svc *Service, r io.Reader) (*ImportResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	cols := make(map[string]int, len(header))
	for i, h := range header {
		cols[strings.ToLower(strings.TrimSpace(h))] = i
	}

	required := []string{"sku", "name", "unit"}
	for _, c := range required {
		if _, ok := cols[c]; !ok {
			return nil, fmt.Errorf("missing required column: %s", c)
		}
	}

	result := &ImportResult{}
	lineNum := 1

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		lineNum++
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: parse error: %v", lineNum, err))
			continue
		}

		get := func(col string) string {
			idx, ok := cols[col]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}

		req := CreateItemRequest{
			SKU:  get("sku"),
			Name: get("name"),
			Unit: get("unit"),
		}
		if req.SKU == "" || req.Name == "" || req.Unit == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: sku, name, unit required", lineNum))
			continue
		}

		if v := get("category"); v != "" {
			req.Category = &v
		}
		if v := get("description"); v != "" {
			req.Description = &v
		}
		if v := get("min_stock_qty"); v != "" {
			d, err := decimal.NewFromString(v)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("line %d: invalid min_stock_qty %q", lineNum, v))
				continue
			}
			req.MinStockQty = &d
		}

		_, err = svc.CreateItem(ctx, req)
		if err != nil {
			if err == ErrDuplicateSKU {
				result.Skipped++
				continue
			}
			result.Errors = append(result.Errors, fmt.Sprintf("line %d (%s): %v", lineNum, req.SKU, err))
			continue
		}
		result.Created++
	}

	return result, nil
}
