package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeUserIds(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty", "", "", false},
		{"whitespace", "   ", "", false},
		{"single", "100", "100", false},
		{"comma list", "100,200,300", "100,200,300", false},
		{"with spaces", " 100 , 200 ", "100,200", false},
		{"dedup", "100,200,100", "100,200", false},
		{"trailing comma", "100,200,", "100,200", false},
		{"invalid id", "100,abc,300", "", true},
		{"zero id rejected", "100,0", "", true},
		{"negative id rejected", "100,-5", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &model.Channel{UserIds: tt.input}
			err := normalizeUserIds(ch)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.input, ch.UserIds, "input should be unchanged on error")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, ch.UserIds)
		})
	}
}

func TestNormalizeUserIdsNilChannel(t *testing.T) {
	require.NoError(t, normalizeUserIds(nil))
}
