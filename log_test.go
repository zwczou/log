package log

import (
	"bytes"
	"testing"
)

func Test_Logger(t *testing.T) {
	buff := bytes.NewBufferString("")
	log := New(buff, "", 0)
	log.Infof("hello world!")
	if buff.String() != "INFO: hello world!\n" {
		log.Fatal("want %s got %s", "INFO: hello world1", buff.String())
	}
}
