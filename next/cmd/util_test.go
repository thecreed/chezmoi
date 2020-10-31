package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpperSnakeCaseToCamelCaseMap(t *testing.T) {
	actual := upperSnakeCaseToCamelCaseMap(map[string]string{
		"BUG_REPORT_URL": "",
		"ID":             "",
	})
	assert.Equal(t, map[string]string{
		"bugReportURL": "",
		"id":           "",
	}, actual)
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}
