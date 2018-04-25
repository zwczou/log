package log

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"
)

func Test_FilePerm(t *testing.T) {
	w := NewFileWriter()
	w.Perm = "0666"
	w.Open("test.log")
	l := New(w, "", LstdFlags)
	l.Info("info")
	l.Debug("debug")
	l.Warn("warn")
	file, err := os.Stat("test.log")
	if err != nil {
		t.Fatal(err)
	}
	if file.Mode() != 0666 {
		t.Fatal("unexpected log file permission")
	}
	os.Remove("test.log")
}

func Test_File1(t *testing.T) {
	w := NewFileWriter()
	w.Open("test.log")
	l := New(w, "", LstdFlags)
	l.SetLevel(Linfo)
	l.Info("info")
	l.Debug("debug")
	l.Warn("warn")

	f, err := os.Open("test.log")
	if err != nil {
		t.Fatal(err)
	}

	b := bufio.NewReader(f)
	lineNum := 0
	for {
		line, _, err := b.ReadLine()
		if err != nil {
			break
		}
		if len(line) > 0 {
			lineNum++
		}
	}
	var expected = 2
	if lineNum != expected {
		t.Fatal(lineNum, "not "+strconv.Itoa(expected)+" lines")
	}
	os.Remove("test.log")
}

func Test_Rotate1(t *testing.T) {
	w := NewFileWriter()
	w.MaxLines = 3
	w.Open("test3.log")

	l := New(w, "", LstdFlags)
	l.Info("info")
	l.Debug("debug")
	l.Warn("warn")
	l.Error("error")
	w.Sync()

	rotateName := "test3" + fmt.Sprintf(".%s.%03d", time.Now().Format("2006-01-02"), 1) + ".log"
	b, err := exists(rotateName)
	if !b || err != nil {
		os.Remove("test3.log")
		t.Fatal("rotate not generated")
	}
	os.Remove(rotateName)
	os.Remove("test3.log")
}

func Test_Rotate2(t *testing.T) {
	w := NewFileWriter()
	w.MaxSize = 100
	w.Open("test2.log")

	l := New(w, "", LstdFlags)
	l.Info("info")
	l.Info("info")
	l.Info("info")
	l.Debug("debug")
	l.Warn("warn")
	l.Error("error")
	w.Sync()

	rotateName := "test2" + fmt.Sprintf(".%s.%03d", time.Now().Format("2006-01-02"), 1) + ".log"
	b, err := exists(rotateName)
	if !b || err != nil {
		os.Remove("test2.log")
		t.Fatal("rotate not generated")
	}
	os.Remove(rotateName)
	os.Remove("test2.log")
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func testFileDailyRotate(t *testing.T, fn1, fn2 string) {
	w := NewFileWriter()
	w.Open(fn1)
	w.dailyOpenTime = time.Now().Add(-24 * time.Hour)
	w.dailyOpenDate = w.dailyOpenTime.Day()
	today, _ := time.ParseInLocation("2006-01-02", time.Now().Format("2006-01-02"), w.dailyOpenTime.Location())
	today = today.Add(-1 * time.Second)
	w.dailyRotate(today)
	for _, file := range []string{fn1, fn2} {
		_, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		content, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if len(content) > 0 {
			t.FailNow()
		}
		os.Remove(file)
	}
	w.Close()
}

func Test_Rotate3(t *testing.T) {
	fn1 := "rotate_day.log"
	fn2 := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".log"
	testFileDailyRotate(t, fn1, fn2)
}

func Test_Rotate4(t *testing.T) {
	fn1 := "rotate_day.log"
	fn := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".log"
	os.Create(fn)
	fn2 := "rotate_day." + time.Now().Add(-24*time.Hour).Format("2006-01-02") + ".001.log"
	testFileDailyRotate(t, fn1, fn2)
	os.Remove(fn)
}
