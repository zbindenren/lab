package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ackerr/lab/internal"
)

func TestGroupCountsReport(t *testing.T) {
	tests := []struct {
		name   string
		counts []internal.GroupProjects
		want   string
	}{
		{
			name: "aligns on the longest group path",
			counts: []internal.GroupProjects{
				{Group: "kubernetes", Projects: 8},
				{Group: "linux", Projects: 42},
			},
			want: "kubernetes 8\nlinux      42\n",
		},
		{
			name:   "group without projects",
			counts: []internal.GroupProjects{{Group: "linux/empty"}},
			want:   "linux/empty 0\n",
		},
		{
			name: "no groups",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, groupCountsReport(tt.counts))
		})
	}
}
