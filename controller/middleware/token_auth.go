package middleware

import (
	"github.com/dimfeld/httptreemux"
	"github.com/fernet/fernet-go"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/common"
	"github.com/ssrs100/blueserver/sesscache"
	"net/http"
)

func Auth(fn httptreemux.HandlerFunc) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ps map[string]string) {
		c, err := r.Cookie(common.CookieSessionId)
		if err != nil {
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}
		logs.Info("sessionName:%s", c.Name)
		logs.Info("session:%s", c.Value)
		k := sesscache.Get(c.Value)
		if len(k) == 0 {
			redirectAddr := conf.GetString("redirect_addr")
			http.Redirect(w, r, redirectAddr, http.StatusFound)
			return
		}
		logs.Info("key:%s", k)
		keys := fernet.MustDecodeKeys(k)
		tokenStr := fernet.VerifyAndDecrypt([]byte(c.Value), 0, keys)
		if len(tokenStr) == 0 {
			sesscache.Del(c.Value)
			http.Error(w, http.StatusText(401), http.StatusUnauthorized)
			return
		}
		sesscache.Touch(c.Value)
		logs.Info("tokenStr:%s", tokenStr)
		fn(w, r, ps)
	}
}
