package utils

import (
	"errors"
	"fmt"
	"os"
	"regexp"
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
		return errors.New(fmt.Sprintf("%v already exists and is not a directory", path))
	} else if stat.Mode() != perms {
		// Dir exists, but wrong perms, so chmod
		return os.Chmod(path, (stat.Mode() & perms))
	}
	return nil
}

type FileSize uint64

var fsregex = regexp.MustCompile(`(\d+\.?\d*)(tb|gb|mb|kb|b)?`)

const (
	KB FileSize = 1000
	MB FileSize = 1000 * KB
	GB FileSize = 1000 * MB
	TB FileSize = 1000 * GB
)

func ParseFileSize(s string) (FileSize, error) {
	var fs FileSize
	err := fs.UnmarshalText([]byte(s))
	return fs, err
}

func (s FileSize) MarshalText() ([]byte, error) {
	if s > TB {
		return []byte(fmt.Sprintf("%.2ftb", float64(s)/float64(TB))), nil
	} else if s > GB {
		return []byte(fmt.Sprintf("%.2fgb", float64(s)/float64(GB))), nil
	} else if s > MB {
		return []byte(fmt.Sprintf("%.2fmb", float64(s)/float64(MB))), nil
	} else if s > KB {
		return []byte(fmt.Sprintf("%.2fkb", float64(s)/float64(KB))), nil
	}
	return []byte(fmt.Sprintf("%db", s)), nil
}

func (s *FileSize) UnmarshalText(bs []byte) error {
	matches := fsregex.FindAllStringSubmatch(strings.ToLower(string(bs)), -1)
	if len(matches) != 1 {
		return errors.New(fmt.Sprintf(`bad filesize: "%v"`, string(bs)))
	} else if len(matches[0]) != 3 {
		return errors.New(fmt.Sprintf(`bad filesize: "%v"`, string(bs)))
	}
	var (
		num  = matches[0][1]
		unit = matches[0][2]
	)
	bytes, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return err
	}

	switch unit {
	case "", "b":
	case "kb":
		bytes *= float64(KB)
	case "mb":
		bytes *= float64(MB)
	case "gb":
		bytes *= float64(GB)
	case "tb":
		bytes *= float64(TB)
	default:
		return errors.New(fmt.Sprint(`bad filesize unit: "%v"`, unit))
	}
	*s = FileSize(bytes)
	return nil
}

func (s FileSize) String() string {
	str, _ := s.MarshalText()
	return string(str)
}
