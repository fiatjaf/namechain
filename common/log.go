package common

import (
	"os"

	"github.com/rs/zerolog"
)

var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
