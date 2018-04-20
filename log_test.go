package log

import (
	"bytes"
	"testing"
)

func Test_Logger(t *testing.T) {
	buff := bytes.NewBufferString("")
	log := New(buff, "", 0)

	buff.Reset()
	log.Debugf("debug")
	if buff.String() != "DEBUG: debug\n" {
		t.Fatalf("want %s got %s", "DEBUG: debug", buff.String())
	}

	buff.Reset()
	log.Infof("hello world!")
	if buff.String() != "INFO: hello world!\n" {
		t.Fatalf("want %s got %s", "INFO: hello world1", buff.String())
	}

	buff.Reset()
	log.Warnf("warn")
	if buff.String() != "WARN: warn\n" {
		t.Fatalf("want %s got %s", "WARN: warn", buff.String())
	}

	buff.Reset()
	log.Errorf("e")
	if buff.String() != "ERROR: e\n" {
		t.Fatalf("want %s got %s", "ERROR: e", buff.String())
	}
}
