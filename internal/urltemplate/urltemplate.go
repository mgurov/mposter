package urltemplate

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type RowToString func(row []string) (string, error)

func Parse(input string) (RowToString, error) {

	position := 0

	parts := []RowToString{}

	for position < len(input) {
		nextPlaceholderSubStart := strings.Index(input[position:], "{{")
		if -1 == nextPlaceholderSubStart {
			//reached the end of the string without new placeholder - append the remainder
			remainder := input[position:]
			if "" != remainder {
				parts = append(parts, constant(remainder))
			}
			break
		}

		if nextPlaceholderSubStart > 0 {
			//something before the placeholder - prepend it
			parts = append(parts, constant(input[position:position+nextPlaceholderSubStart]))
		}

		placeholderFun, nextPosition, err := scanPlaceholder(input[position+nextPlaceholderSubStart:])
		if err != nil {
			return nil, err
		}
		parts = append(parts, placeholderFun)
		position += nextPlaceholderSubStart + nextPosition
	}

	return func(input []string) (string, error) {
		result := bytes.Buffer{}
		for _, p := range parts {
			partString, err := p(input)
			if err != nil {
				return "", err
			}
			result.WriteString(partString)
		}
		return result.String(), nil
	}, nil
}

func scanPlaceholder(input string) (RowToString, int, error) {
	placeholderEnd := strings.Index(input, "}}")
	if -1 == placeholderEnd {
		return nil, -1, fmt.Errorf("placeholder '%s' isn't terminated", input)
	}
	placeholderContent := input[2:placeholderEnd]
	placeholderFun, err := buildPlaceholderFun(placeholderContent)
	return placeholderFun, placeholderEnd + 2, err
}

func buildPlaceholderFun(placeholderContent string) (RowToString, error) {
	trimmed := strings.TrimSpace(placeholderContent)
	index, err := strconv.Atoi(trimmed)
	if nil != err {
		return nil, fmt.Errorf("placeholder '{{%s}}' isn't recognized", placeholderContent)
	}

	return func(row []string) (string, error) {
		if len(row)-1 < index {
			return "", fmt.Errorf("data missing for placeholder {{%s}}", placeholderContent)
		}

		return row[index], nil
	}, nil

}

func constant(input string) RowToString {
	return func(_ []string) (string, error) { return input, nil }
}
