package cli

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	defaultPaginateFlagValue = "0:200"
	maxPageSize              = 200
)

type paginateSpec struct {
	From  int
	Limit int
	All   bool
}

func parsePaginateValue(raw string) (paginateSpec, error) {
	var zero paginateSpec
	value := strings.TrimSpace(raw)
	if value == "" {
		return zero, invalidUsage("invalid --paginate: value cannot be empty")
	}

	if strings.EqualFold(value, "all") {
		return paginateSpec{All: true}, nil
	}

	fromRaw, limitRaw, hasRange := strings.Cut(value, ":")
	if hasRange {
		from, err := parseOptionalFromValue(strings.TrimSpace(fromRaw), raw)
		if err != nil {
			return zero, err
		}
		limitSpec, err := parseLimitValue(strings.TrimSpace(limitRaw), from, raw)
		if err != nil {
			return zero, err
		}
		return limitSpec, nil
	}

	limit, err := parsePositiveInt("paginate limit", value)
	if err != nil {
		return zero, invalidUsage("invalid --paginate %q: %v", raw, err)
	}
	return paginateSpec{Limit: limit}, nil
}

func parseOptionalFromValue(fromRaw, raw string) (int, error) {
	if fromRaw == "" {
		return 0, nil
	}
	from, err := parseNonNegativeInt("paginate from", fromRaw)
	if err != nil {
		return 0, invalidUsage("invalid --paginate %q: %v", raw, err)
	}
	return from, nil
}

func parseLimitValue(limitRaw string, from int, raw string) (paginateSpec, error) {
	if strings.EqualFold(limitRaw, "all") {
		return paginateSpec{From: from, All: true}, nil
	}
	limit, err := parsePositiveInt("paginate limit", limitRaw)
	if err != nil {
		return paginateSpec{}, invalidUsage("invalid --paginate %q: %v", raw, err)
	}
	return paginateSpec{From: from, Limit: limit}, nil
}

func parseNonNegativeInt(label, raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", label)
	}
	if value < 0 {
		return 0, fmt.Errorf("%s must be >= 0", label)
	}
	return value, nil
}

func parsePositiveInt(label, raw string) (int, error) {
	value, err := parseNonNegativeInt(label, raw)
	if err != nil {
		return 0, err
	}
	if value == 0 {
		return 0, fmt.Errorf("%s must be > 0", label)
	}
	return value, nil
}

func collectPageRange[T any](spec paginateSpec, fetch func(page, size int) ([]T, error)) ([]T, error) {
	if spec.All && spec.From == 0 {
		return collectAllPages(fetch)
	}

	startPage := spec.From / maxPageSize
	skipInFirstPage := spec.From % maxPageSize

	collected := make([]T, 0)
	page := startPage
	for {
		items, err := fetch(page, maxPageSize)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return collected, nil
		}

		current := items
		if page == startPage && skipInFirstPage > 0 {
			if skipInFirstPage >= len(current) {
				if len(items) < maxPageSize {
					return collected, nil
				}
				page++
				continue
			}
			current = current[skipInFirstPage:]
		}

		if spec.All {
			collected = append(collected, current...)
		} else {
			remaining := spec.Limit - len(collected)
			if remaining <= 0 {
				return collected, nil
			}
			if len(current) > remaining {
				current = current[:remaining]
			}
			collected = append(collected, current...)
			if len(collected) >= spec.Limit {
				return collected, nil
			}
		}

		if len(items) < maxPageSize {
			return collected, nil
		}
		page++
	}
}

func collectAllPages[T any](fetch func(page, size int) ([]T, error)) ([]T, error) {
	collected := make([]T, 0)
	page := 0
	for {
		items, err := fetch(page, maxPageSize)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return collected, nil
		}
		collected = append(collected, items...)
		if len(items) < maxPageSize {
			return collected, nil
		}
		page++
	}
}

func sliceByPaginate[T any](items []T, spec paginateSpec) []T {
	if spec.From >= len(items) {
		return []T{}
	}
	start := spec.From
	if spec.All {
		return items[start:]
	}
	end := start + spec.Limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
