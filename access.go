// Debugger

package ogo

import (
	"encoding/json"
	"net/http"
	"time"
	//"strings"

	//"github.com/Odinman/ogo/libs/logs"
)

type Access struct {
	Time     time.Time   `json:"t"`
	Service  string      `json:"sn,omitempty"`
	Session  string      `json:"s"`
	Duration string      `json:"d"`
	Http     *HTTPLog    `json:"http,omitempty"`
	App      *AppLog     `json:"app,omitempty"`   //rest app日志
	Debug    interface{} `json:"debug,omitempty"` //app debug日志
}

type HTTPLog struct {
	Status    int          `json:"sc"`
	IP        string       `json:"ip"`
	Method    string       `json:"m"`
	URI       string       `json:"uri"`
	Proto     string       `json:"p"`
	ReqBody   string       `json:"qb,omitempty"`
	ReqLength int          `json:"ql"` //请求body大小
	RepLength int          `json:"pl"` //返回body大小
	Host      string       `json:"h"`
	InHeader  *http.Header `json:"ih,omitempty"`
	OutHeader http.Header  `json:"oh,omitempty"`
}

type AppLog struct {
	Ctag   string      `json:"ctag"`
	Query  interface{} `json:"query,omitempty"`
	New    interface{} `json:"new,omitempty"`
	Old    interface{} `json:"old,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

/* {{{ func NewAccess() *Access
 *
 */
func NewAccess() *Access {
	ac := new(Access) //access日志信息
	ac.Time = time.Now()
	//ac.App = new(AppLog)
	return ac
}

/* }}} */

/* {{{ func (ac *Access) Save()
 * 记录access日志
 */
func (ac *Access) Save() {
	if ab, err := json.Marshal(ac); err == nil {
		accessor.Access(string(ab))
	}
}

/* }}} */

/* {{{ func (ac *Access) SaveApp(al *AppLog)
 * 放置app日志
 */
func (ac *Access) SaveApp(al *AppLog) {
	ac.App = al
}

/* }}} */

/* {{{ func (ac *Access) GetApp() interface{}
 * 放置app日志
 */
func (ac *Access) GetApp() interface{} {
	return ac.App
}

/* }}} */

/* {{{ func (ac *Access) SaveDebug(i interface{})
 * 放置app日志
 */
func (ac *Access) SaveDebug(i interface{}) {
	ac.Debug = i
}

/* }}} */
