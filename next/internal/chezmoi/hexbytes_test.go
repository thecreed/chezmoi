package chezmoi

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexBytes(t *testing.T) {
	for i, tc := range []struct {
		b           hexBytes
		expectedStr string
	}{
		{
			b:           nil,
			expectedStr: `""`,
		},
		{
			b:           []byte{0},
			expectedStr: `"00"`,
		},
		{
			b:           []byte{0, 1, 2, 3},
			expectedStr: `"00010203"`,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual, err := json.Marshal(tc.b)
			require.NoError(t, err)
			assert.Equal(t, []byte(tc.expectedStr), actual)
			var actualB hexBytes
			require.NoError(t, json.Unmarshal(actual, &actualB))
			assert.Equal(t, tc.b, actualB)
		})
	}
}
