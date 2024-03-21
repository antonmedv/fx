package engine

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"regexp"
)

func isFile(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func open(filePath string, flagYaml *bool) *os.File {
	f, err := os.Open(filePath)
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			println(err.Error())
			os.Exit(1)
		} else {
			panic(err)
		}
	}
	fileName := path.Base(filePath)
	hasYamlExt, _ := regexp.MatchString(`(?i)\.ya?ml$`, fileName)
	if !*flagYaml && hasYamlExt {
		*flagYaml = true
	}
	return f
}
