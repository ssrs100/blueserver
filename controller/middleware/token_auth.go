package middleware

import (
	"encoding/json"
	"github.com/dimfeld/httptreemux"
	"github.com/fernet/fernet-go"
	"github.com/jack0liu/conf"
	"github.com/jack0liu/logs"
	"github.com/ssrs100/blueserver/common"
	"github.com/ssrs100/blueserver/sesscache"
	"net/http"
	"time"
)

type UserSession struct {
	UserId    string   `json:"user_id"`
	Roles     []string `json:"roles"`
	ExpiredAt string   `json:"expired_at"`
	CreatedAt string   `json:"created_at"`
}

func Auth(fn httptreemux.HandlerFunc) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ps map[string]string) {
		//c, err := r.Cookie(common.CookieSessionId)
		//if err != nil {
		//	http.Error(w, http.StatusText(400), http.StatusBadRequest)
		//	return
		//}
		if conf.GetInt("enable_auth") == -1 {
			fn(w, r, ps)
			return
		}
		token := r.Header.Get(common.XAuthB)
		logs.Info("session:%s", token)
		k := sesscache.Get(token)
		if len(k) == 0 {
			redirectAddr := conf.GetString("redirect_addr")
			http.Redirect(w, r, redirectAddr, http.StatusFound)
			return
		}
		logs.Info("key:%s", k)
		keys := fernet.MustDecodeKeys(k)
		tokenStr := fernet.VerifyAndDecrypt([]byte(token), 0, keys)
		if len(tokenStr) == 0 {
			sesscache.Del(token)
			http.Error(w, http.StatusText(401), http.StatusUnauthorized)
			return
		}
		projectId := ps["projectId"]
		if len(projectId) > 0 {
			var us UserSession
			if err := json.Unmarshal(tokenStr, &us); err != nil {
				logs.Error("invalid user session, str:%s", tokenStr)
				http.Error(w, http.StatusText(401), http.StatusUnauthorized)
				return
			}
			if projectId != us.UserId {
				logs.Error("invalid user session, project id(%s) not equal %s, str:%s", projectId, us.UserId, tokenStr)
				http.Error(w, http.StatusText(401), http.StatusUnauthorized)
				return
			}

			sesscache.SetWithNoExpired("lastAccess_"+us.UserId, time.Now().Format(time.RFC3339))
		}

		sesscache.TouchWithExpired(token, time.Hour * 24 * 7)
		logs.Info("tokenStr:%s", tokenStr)
		fn(w, r, ps)
	}
}
