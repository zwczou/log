package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileWriter struct {
	sync.RWMutex

	Filename             string
	Writer               *os.File
	MaxLines             int
	maxLinesCurLines     int
	MaxSize              int
	maxSizeCurSize       int
	MaxDays              int64
	dailyOpenDate        int
	dailyOpenTime        time.Time
	Perm                 string
	RotatePerm           string
	fileNameOnly, suffix string
}

func NewFileWriter() *FileWriter {
	w := &FileWriter{
		MaxDays:    7,
		RotatePerm: "0440",
		Perm:       "0660",
	}
	return w
}

func (w *FileWriter) Open(filename string) error {
	w.Filename = filename
	w.suffix = filepath.Ext(w.Filename)
	w.fileNameOnly = strings.TrimSuffix(w.Filename, w.suffix)
	if w.suffix == "" {
		w.suffix = ".log"
	}
	err := w.startLogger()
	return err
}

func (w *FileWriter) startLogger() error {
	file, err := w.createLogFile()
	if err != nil {
		return err
	}
	if w.Writer != nil {
		w.Writer.Close()
	}
	w.Writer = file
	return w.initFd()
}

func (w *FileWriter) createLogFile() (*os.File, error) {
	perm, err := strconv.ParseInt(w.Perm, 8, 64)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(w.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(perm))
	if err == nil {
		os.Chmod(w.Filename, os.FileMode(perm))
	}
	return fd, err
}

func (w *FileWriter) initFd() error {
	fd := w.Writer
	fInfo, err := fd.Stat()
	if err != nil {
		return fmt.Errorf("get stat err: %s", err)
	}
	w.maxSizeCurSize = int(fInfo.Size())
	w.dailyOpenTime = time.Now()
	w.dailyOpenDate = w.dailyOpenTime.Day()
	w.maxLinesCurLines = 0
	go w.dailyRotate(w.dailyOpenTime)
	if fInfo.Size() > 0 && w.MaxLines > 0 {
		count, err := w.lines()
		if err != nil {
			return err
		}
		w.maxLinesCurLines = count
	}
	return nil
}

func (w *FileWriter) needRotate(size int, day int) bool {
	return (w.MaxLines > 0 && w.maxLinesCurLines >= w.MaxLines) ||
		(w.MaxSize > 0 && w.maxSizeCurSize >= w.MaxSize) ||
		(day != w.dailyOpenDate)
}

func (w *FileWriter) dailyRotate(openTime time.Time) {
	y, m, d := openTime.Add(24 * time.Hour).Date()
	nextDay := time.Date(y, m, d, 0, 0, 0, 0, openTime.Location())
	tm := time.NewTimer(time.Duration(nextDay.UnixNano() - openTime.UnixNano() + 100))
	<-tm.C
	w.Lock()
	if w.needRotate(0, time.Now().Day()) {
		if err := w.doRotate(time.Now()); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
		}
	}
	w.Unlock()
}

func (w *FileWriter) lines() (int, error) {
	fd, err := os.Open(w.Filename)
	if err != nil {
		return 0, err
	}
	defer fd.Close()

	buf := make([]byte, 32768) // 32k
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := fd.Read(buf)
		if err != nil && err != io.EOF {
			return count, err
		}

		count += bytes.Count(buf[:c], lineSep)

		if err == io.EOF {
			break
		}
	}

	return count, nil
}

func (w *FileWriter) doRotate(logTime time.Time) error {
	num := 1
	fName := ""
	rotatePerm, err := strconv.ParseInt(w.RotatePerm, 8, 64)
	if err != nil {
		return err
	}

	_, err = os.Lstat(w.Filename)
	if err != nil {
		goto restart
	}

	if w.MaxLines > 0 || w.MaxSize > 0 {
		for ; err == nil && num <= 999; num++ {
			fName = w.fileNameOnly + fmt.Sprintf(".%s.%03d%s", logTime.Format("2006-01-02"), num, w.suffix)
			_, err = os.Lstat(fName)
		}
	} else {
		fName = fmt.Sprintf("%s.%s%s", w.fileNameOnly, w.dailyOpenTime.Format("2006-01-02"), w.suffix)
		_, err = os.Lstat(fName)
		for ; err == nil && num <= 999; num++ {
			fName = w.fileNameOnly + fmt.Sprintf(".%s.%03d%s", w.dailyOpenTime.Format("2006-01-02"), num, w.suffix)
			_, err = os.Lstat(fName)
		}
	}
	if err == nil {
		return fmt.Errorf("Rotate: Cannot find free log number to rename %s", w.Filename)
	}

	w.Writer.Close()

	err = os.Rename(w.Filename, fName)
	if err != nil {
		goto restart
	}

	err = os.Chmod(fName, os.FileMode(rotatePerm))

restart:

	startLoggerErr := w.startLogger()
	go w.deleteOldLog()

	if startLoggerErr != nil {
		return fmt.Errorf("Rotate StartLogger: %s", startLoggerErr)
	}
	if err != nil {
		return fmt.Errorf("Rotate: %s", err)
	}
	return nil
}

func (w *FileWriter) deleteOldLog() {
	dir := filepath.Dir(w.Filename)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Unable to delete old log '%s', error: %v\n", path, r)
			}
		}()

		if info == nil {
			return
		}

		if !info.IsDir() && info.ModTime().Add(24*time.Hour*time.Duration(w.MaxDays)).Before(time.Now()) {
			if strings.HasPrefix(filepath.Base(path), filepath.Base(w.fileNameOnly)) &&
				strings.HasSuffix(filepath.Base(path), w.suffix) {
				os.Remove(path)
			}
		}
		return
	})
}

func (w *FileWriter) Write(b []byte) (n int, err error) {
	now := time.Now()
	w.RLock()
	need := w.needRotate(len(b), now.Day())
	w.RUnlock()
	if need {
		w.Lock()
		if w.needRotate(len(b), now.Day()) {
			if err := w.doRotate(now); err != nil {
				fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
			}
		}
		w.Unlock()
	}
	n, err = w.Writer.Write(b)
	if err == nil {
		w.maxLinesCurLines++
		w.maxSizeCurSize += len(b)
	}
	return
}

func (w *FileWriter) Close() error {
	return w.Writer.Close()
}

func (w *FileWriter) Sync() error {
	return w.Writer.Sync()
}
