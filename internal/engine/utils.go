package engine

import (
	"math/big"
	"reflect"
	"regexp"
	"time"

	"github.com/dop251/goja"
)

var (
	bigIntType   = reflect.TypeOf((*big.Int)(nil))
	timeTimeType = reflect.TypeOf((*time.Time)(nil)).Elem()
)

var (
	syntaxErrorRe = regexp.MustCompile(`^SyntaxError: SyntaxError: \(anonymous\): Line \d+:\d+\s+`)
	andMoreErrors = regexp.MustCompile(`\(and \d+ more errors\)$`)
)

func extractErrorMessage(s string) string {
	s = syntaxErrorRe.ReplaceAllString(s, "")
	s = andMoreErrors.ReplaceAllString(s, "")
	return s
}

func errorToString(err error) string {
	if exception, ok := err.(*goja.Exception); ok {
		message := exception.Value().String()
		message = extractErrorMessage(message)
		return message
	}
	return err.Error()
}
