// +build daemon

package ogo

/* {{{ import
 */
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"github.com/VividCortex/godaemon"
	"github.com/nightlyone/lockfile"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web/middleware"
)

/* }}} */

/* {{{ func Run()
 * Run ogo application.
 */
func Run() {
	defer func() {
		if err := recover(); err != nil {
			WriteMsg("App crashed with error:", err)
			for i := 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				WriteMsg(file, line)
			}
			//panic要输出到console
			fmt.Println("App crashed with error:", err)
		}
	}()
	if Env.Daemonize {
		godaemon.MakeDaemon(&godaemon.DaemonAttr{})
	}
	//check&write pidfile, added by odin
	dir := filepath.Dir(Env.PidFile)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			//mkdir
			if err := os.Mkdir(dir, 0755); err != nil {
				panic(err)
			}
		}
	}
	if l, err := lockfile.New(Env.PidFile); err == nil {
		if le := l.TryLock(); le != nil {
			panic(le)
		}
	} else {
		panic(err)
	}

	var mainErr error

	Debugger.Debug("will run worker: %v", Env.Worker)
	if worker, ok := Ctx.Workers[Env.Worker]; ok {
		vw := reflect.New(worker.WorkerType)
		execWorker, ok := vw.Interface().(WorkerInterface)
		if !ok {
			panic("worker is not WorkerInterface")
		}

		//Init
		execWorker.Init(Ctx, Env.Worker)

		//Main
		mainErr = execWorker.Main()
	} else {
		mainErr = errors.New("not found worker: " + Env.Worker)
	}

	if mainErr != nil {
		panic(mainErr)
	}
	//睡一段时间再结束
	time.Sleep(1000 * time.Microsecond)
}

/* }}} */