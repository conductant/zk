package template

import (
	"errors"
	"fmt"
)

var (
	ErrMissingTemplateFunc = errors.New("err-no-template-func")
	ErrBadTemplateFunc     = errors.New("err-bad-template-func")
	ErrNotImplemented      = errors.New("err-not-implemented")
)

type NotSupported struct {
	Protocol string
}

func (this *NotSupported) Error() string {
	return fmt.Sprintf("err-template-not-supported: %s", this.Protocol)
}
