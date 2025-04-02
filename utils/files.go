package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// EnsureDirAndMaxPerms ensures that the given path exists, that it's a directory,
// and that it has permissions that are no more permissive than the given ones.
//
// - If the path does not exist, it is created
// - If the path exists, but is not a directory, an error is returned
// - If the path exists, and is a directory, but has the wrong perms, it is chmod'ed
func EnsureDirAndMaxPerms(path string, perms os.FileMode) error {
	stat, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		// Regular error
		return err
	} else if os.IsNotExist(err) {
		// Dir doesn't exist, create it with desired perms
		return os.MkdirAll(path, perms)
	} else if !stat.IsDir() {
		// Path exists, but it's a file, so don't clobber
		return fmt.Errorf("%v already exists and is not a directory", path)
	} else if stat.Mode() != perms {
		// Dir exists, but wrong perms, so chmod
		return os.Chmod(path, (stat.Mode() & perms))
	}
	return nil
}

type ByteSize int64

const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
)

func (b *ByteSize) UnmarshalText(text []byte) error {
	str := strings.TrimSpace(strings.ToLower(string(text)))

	var multiplier ByteSize = 1
	var suffix string

	// More efficient suffix extraction
	if len(str) > 0 {
		lastChar := str[len(str)-1]
		if lastChar == 'b' {
			if len(str) > 2 && str[len(str)-2] != 'p' {
				suffix = str[len(str)-2:]
				str = str[:len(str)-2]
			} else {
				suffix = str[len(str)-1:]
				str = str[:len(str)-1]
			}
		}
	}

	switch suffix {
	case "kb":
		multiplier = KB
	case "mb":
		multiplier = MB
	case "gb":
		multiplier = GB
	case "tb":
		multiplier = TB
	case "pb":
		multiplier = PB
	}

	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fmt.Errorf("invalid byte size: %s", string(text))
	}

	*b = ByteSize(value * float64(multiplier))
	return nil
}

func (b ByteSize) String() string {
	switch {
	case b >= PB:
		return fmt.Sprintf("%.2fPB", float64(b)/float64(PB))
	case b >= TB:
		return fmt.Sprintf("%.2fTB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.2fGB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2fMB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2fKB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
