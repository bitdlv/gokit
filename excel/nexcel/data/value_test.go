package data

import (
	"errors"
	"testing"

	"github.com/bitdlv/gokit/errx"
)

func TestXxx(t *testing.T) {
	err := errx.New(1, "")

	e := errx.Error{}

	println(errors.Is(err, &e))
	// ok := errors.As(err, &e)
	// require.True(t, ok)
	// fmt.Printf("%+v\n", e)
}
