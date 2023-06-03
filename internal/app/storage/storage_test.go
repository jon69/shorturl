package storage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	require.NotNil(t, NewStorage("", ""))
}

func TestPutURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "First reference",
			value: "http://yandex.ru",
			want:  "1",
		},
		{
			name:  "Second reference",
			value: "http:/google.com",
			want:  "2",
		},
	}

	st := NewStorage("", "")
	require.NotNil(t, st)

	for _, tt := range tests {
		_, val := st.PutURL(tt.value)
		assert.Equal(t, tt.want, val)
	}
}

func TestGetURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "First reference",
			value: "http://yandex.ru",
			want:  "1",
		},
		{
			name:  "Second reference",
			value: "http:/google.com",
			want:  "2",
		},
	}

	storage := NewStorage("", "")
	require.NotNil(t, storage)

	for _, tt := range tests {
		_, v := storage.PutURL(tt.value)
		assert.Equal(t, tt.want, v)

		url, ok, deleted := storage.GetURL(tt.want)
		assert.Equal(t, ok, true)
		assert.Equal(t, deleted, false)
		assert.Equal(t, url, tt.value)
	}
}

func BenchmarkGetURL(b *testing.B) {
	storage := NewStorage("", "")
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("http://%s_%d.ru", "yandex", i)
		storage.PutURL(s)
	}
}
