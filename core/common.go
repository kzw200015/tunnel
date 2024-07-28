package core

import (
	"github.com/pkg/errors"
	"io"
	"log"
)

func CloseAndLog(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Println(errors.WithStack(err))
	}
}

func CopyStream(dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Println(errors.WithStack(err))
	}
}

func LogWithStack(err error) {
	log.Printf("%+v", errors.WithStack(err))
}
