package logs

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileLogWriter implements LoggerInterface.
// It writes messages by lines limit, file size limit, or time frequency.
type FileLogWriter struct {
	*log.Logger
	mw *MuxWriter
	// The opened file
	Filename string `json:"filename"`

	// Rotate at size
	Maxsize         int `json:"maxsize"`
	maxsize_cursize int

	// Rotate daily
	Daily          bool  `json:"daily"`
	Maxdays        int64 `json:"maxdays"`
	daily_opendate int

	Rotate bool `json:"rotate"`

	startLock sync.Mutex // Only one log can write to the file

	Level int `json:"level"`
}

// an *os.File writer with locker.
type MuxWriter struct {
	sync.Mutex
	fd *os.File
}

// write to os.File.
func (l *MuxWriter) Write(b []byte) (int, error) {
	l.Lock()
	defer l.Unlock()
	return l.fd.Write(b)
}

// set os.File in writer.
func (l *MuxWriter) SetFd(fd *os.File) {
	if l.fd != nil {
		l.fd.Close()
	}
	l.fd = fd
}

// create a FileLogWriter returning as LoggerInterface.
func NewFileWriter() LoggerInterface {
	w := &FileLogWriter{
		Filename: "",
		Maxsize:  1 << 27, //128 MB
		Daily:    true,
		Maxdays:  3,
		Rotate:   true,
		Level:    LevelTrace,
	}
	// use MuxWriter instead direct use os.File for lock write when rotate
	w.mw = new(MuxWriter)
	// set MuxWriter as Logger's io.Writer
	w.Logger = log.New(w.mw, "", log.Ldate|log.Ltime)
	return w
}

// Init file logger with json config.
// jsonconfig like:
//	{
//	"filename":"logs/beego.log",
//	"maxlines":10000,
//	"maxsize":1<<30,
//	"daily":true,
//	"maxdays":15,
//	"rotate":true
//	}
func (w *FileLogWriter) Init(jsonconfig string) error {
	err := json.Unmarshal([]byte(jsonconfig), w)
	if err != nil {
		return err
	}
	if len(w.Filename) == 0 {
		return errors.New("jsonconfig must have filename")
	}
	err = w.startLogger()
	return err
}

// start file logger. create log file and set to locker-inside file writer.
func (w *FileLogWriter) startLogger() error {
	fd, err := w.createLogFile()
	if err != nil {
		return err
	}
	w.mw.SetFd(fd)
	err = w.initFd()
	if err != nil {
		return err
	}
	return nil
}

func (w *FileLogWriter) docheck(size int) {
	w.startLock.Lock()
	defer w.startLock.Unlock()
	if w.Rotate && ((w.Maxsize > 0 && w.maxsize_cursize >= w.Maxsize) || (w.Daily && time.Now().Day() != w.daily_opendate)) {
		if err := w.DoRotate(); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
			return
		}
	}
	w.maxsize_cursize += size
}

// write logger message into file.
func (w *FileLogWriter) WriteMsg(msg string, level int) error {
	if level < w.Level {
		return nil
	}
	n := 24 + len(msg) // 24 stand for the length "2013/06/23 21:00:22 [T] "
	w.docheck(n)
	w.Logger.Println(msg)
	return nil
}

func (w *FileLogWriter) createLogFile() (*os.File, error) {
	dir := filepath.Dir(w.Filename)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			//mkdir
			if err := os.Mkdir(dir, 0755); err != nil {
				return nil, err
			}
		}
	}
	// Open the log file
	fd, err := os.OpenFile(w.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	return fd, err
}

func (w *FileLogWriter) initFd() error {
	fd := w.mw.fd
	finfo, err := fd.Stat()
	if err != nil {
		return fmt.Errorf("get stat err: %s\n", err)
	}
	w.maxsize_cursize = int(finfo.Size())
	w.daily_opendate = time.Now().Day()
	return nil
}

// DoRotate means it need to write file in new file.
// new file name like xx.log.2013-01-01.2
func (w *FileLogWriter) DoRotate() error {
	_, err := os.Lstat(w.Filename)
	if err == nil { // file exists
		// Find the next available number
		num := 1
		fname := ""
		for ; err == nil && num <= 999; num++ {
			fname = w.Filename + fmt.Sprintf(".%s.%03d", time.Now().Format("2006-01-02"), num)
			_, err = os.Lstat(fname)
		}
		// return error if the last file checked still existed
		if err == nil {
			return fmt.Errorf("Rotate: Cannot find free log number to rename %s\n", w.Filename)
		}

		// block Logger's io.Writer
		w.mw.Lock()
		defer w.mw.Unlock()

		fd := w.mw.fd
		fd.Close()

		// close fd before rename
		// Rename the file to its newfound home
		err = os.Rename(w.Filename, fname)
		if err != nil {
			return fmt.Errorf("Rotate: %s\n", err)
		}

		// re-start logger
		err = w.startLogger()
		if err != nil {
			return fmt.Errorf("Rotate StartLogger: %s\n", err)
		}

		go w.deleteOldLog()
	}

	return nil
}

func (w *FileLogWriter) deleteOldLog() {
	dir := filepath.Dir(w.Filename)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && info.ModTime().Unix() < (time.Now().Unix()-60*60*24*w.Maxdays) {
			if strings.HasPrefix(filepath.Base(path), filepath.Base(w.Filename)) {
				os.Remove(path)
			}
		}
		return nil
	})
}

// destroy file logger, close file writer.
func (w *FileLogWriter) Destroy() {
	w.mw.fd.Close()
}

// flush file logger.
// there are no buffering messages in file logger in memory.
// flush file means sync file from disk.
func (w *FileLogWriter) Flush() {
	w.mw.fd.Sync()
}

func init() {
	Register("file", NewFileWriter)
}
