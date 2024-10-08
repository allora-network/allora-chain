package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/utils"
)

func TestEnsureDirAndMaxPerms(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		perms   os.FileMode
		setup   func(string) error
		wantErr bool
	}{
		{
			name:    "Create new directory",
			path:    filepath.Join(tempDir, "newdir"),
			perms:   0755,
			wantErr: false,
		},
		{
			name:  "Existing directory with correct permissions",
			path:  filepath.Join(tempDir, "existingdir"),
			perms: 0755,
			setup: func(path string) error {
				return os.Mkdir(path, 0755)
			},
			wantErr: false,
		},
		{
			name:  "Existing directory with more permissive permissions",
			path:  filepath.Join(tempDir, "morepermissivedir"),
			perms: 0755,
			setup: func(path string) error {
				return os.Mkdir(path, 0777)
			},
			wantErr: false,
		},
		{
			name:  "Existing file instead of directory",
			path:  filepath.Join(tempDir, "existingfile"),
			perms: 0755,
			setup: func(path string) error {
				return os.WriteFile(path, []byte("test"), 0644) //nolint:gosec
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.setup != nil {
				err := tt.setup(tt.path)
				require.NoError(t, err, "Setup failed")
			}

			err := utils.EnsureDirAndMaxPerms(tt.path, tt.perms)

			if tt.wantErr {
				require.Error(t, err, "Expected an error, but got none")
			} else {
				require.NoError(t, err, "Unexpected error")
			}

			if err == nil {
				info, statErr := os.Stat(tt.path)
				require.NoError(t, statErr, "Failed to stat path")
				require.True(t, info.IsDir(), "Expected %s to be a directory", tt.path)
				require.Equal(t, tt.perms, info.Mode().Perm(), "Expected permissions %v, got %v", tt.perms, info.Mode().Perm())
			}
		})
	}
}

func TestByteSize_UnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    utils.ByteSize
		wantErr bool
	}{
		{"Bytes", "1024b", 1024, false},
		{"Kilobytes", "1kb", 1024, false},
		{"Megabytes", "1mb", 1024 * 1024, false},
		{"Gigabytes", "1gb", 1024 * 1024 * 1024, false},
		{"Terabytes", "1tb", 1024 * 1024 * 1024 * 1024, false},
		{"Petabytes", "1pb", 1024 * 1024 * 1024 * 1024 * 1024, false},
		{"Decimal", "1.5gb", 1610612736, false},
		{"No suffix", "1024", 1024, false},
		{"Uppercase", "1KB", 1024, false},
		{"Invalid suffix", "1ab", 0, true},
		{"Invalid number", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var b utils.ByteSize
			err := b.UnmarshalText([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, b)
		})
	}
}

func TestByteSize_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    utils.ByteSize
		want string
	}{
		{"Bytes", 900, "900B"},
		{"Kilobytes", 1024, "1.00KB"},
		{"Megabytes", 1024 * 1024, "1.00MB"},
		{"Gigabytes", 1024 * 1024 * 1024, "1.00GB"},
		{"Terabytes", 1024 * 1024 * 1024 * 1024, "1.00TB"},
		{"Petabytes", 1024 * 1024 * 1024 * 1024 * 1024, "1.00PB"},
		{"Decimal Gigabytes", 1610612736, "1.50GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.b.String())
		})
	}
}
