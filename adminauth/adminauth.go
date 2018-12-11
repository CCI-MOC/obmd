package adminauth

import (
	"net/http"

	"github.com/CCI-MOC/obmd/token"
	"github.com/gorilla/mux"
)

// Make a subrouter for admin-only requests. This checks for a token passed in
// via basic auth. If the username is not admin or the token does not match
// the argument, none of the routes registered on the returned router will match,
// instead returning 404 (Not found). TODO: think about whether we want that
// as an explicit security feature. It masks the presence or abscence of nodes,
// which is nice (but if we're to rely on that, we need to mitigate timing
// attacks).
func AdminRouter(tok token.Token, r *mux.Router) *mux.Router {
	return r.MatcherFunc(func(req *http.Request, m *mux.RouteMatch) bool {
		user, pass, ok := req.BasicAuth()
		if !(ok && user == "admin") {
			return false
		}
		var reqTok token.Token
		err := (&reqTok).UnmarshalText([]byte(pass))
		if err != nil {
			return false
		}
		return tok.Verify(reqTok) == nil
	}).Subrouter()
}
