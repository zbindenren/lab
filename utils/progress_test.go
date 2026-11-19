package utils

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProgress_Step(t *testing.T) {
	tests := []struct {
		name  string
		steps []string
		quiet bool
		want  string
	}{
		{
			name:  "counts and redraws in place",
			steps: []string{"linux", "linux/foo"},
			want:  "\r\033[K1/2 linux\r\033[K2/2 linux/foo\n",
		},
		{
			name:  "silent without a terminal",
			steps: []string{"linux"},
			quiet: true,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			progress := &Progress{out: &buf, total: len(tt.steps), quiet: tt.quiet}

			for _, step := range tt.steps {
				progress.Step(step)
			}

			progress.Done()
			require.Equal(t, tt.want, buf.String())
		})
	}
}

func TestProgress_StepConcurrent(t *testing.T) {
	var buf bytes.Buffer

	progress := &Progress{out: &buf, total: 50}

	var wg sync.WaitGroup

	for range 50 {

		wg.Go(func() {

			progress.Step("group")
		})
	}

	wg.Wait()
	progress.Done()

	require.Contains(t, buf.String(), "50/50 group")
}
