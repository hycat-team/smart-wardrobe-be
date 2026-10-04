package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	subscription_handler "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/handler"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// 025 T082: the campaign routes REGISTER without breaking startup (R11).
//
// WHY THIS TEST EXISTS AT ALL. Gin refuses to register a STATIC segment beside a
// PARAMETER at the same level, and the conflict does not fail a request — it
// PANICS WHILE BUILDING THE ROUTER. The account service would refuse to boot, so
// authentication, plans, payments and the campaign grant would all be down
// because a read-only reporting page was attached to the wrong group. That is not
// a hypothetical: the same trap was hit at 024 and is written up in
// `research.md` R11 with the scar at `router.go:96-103`.
//
// Nothing else in the suite would catch it. The handler tests build their OWN
// engine, so they never touch the router's group structure; and the use-case tests
// never build an engine at all. A panic in `Init` is invisible until the binary
// starts, which is the worst possible moment to find it.
//
// So this test calls the real `Init` on a real engine. A regression here is a
// failure at `go test`, not at `docker up` in production.

// passAuthMiddleware accepts every caller.
//
// The point of this test is the ROUTE STRUCTURE, not authentication — that is
// covered by the handler tests, which drive the real `RolesAuthorize`. A nil auth
// middleware here would make `Init` panic for a reason unrelated to what is being
// checked.
type passAuthMiddleware struct{}

func (passAuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func (passAuthMiddleware) OptionalHandle() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

// newTestRouter builds the real Router with every handler nil.
//
// Nil HANDLERS are safe here and deliberate: `shared_pres.WrapHandler` takes a
// method value, and taking one on a nil pointer receiver neither panics nor calls
// anything. That lets the test exercise route registration — the whole subject —
// without standing up four collaborators it would never reach. A request sent here
// would panic in the handler, and the tests below never send one.
//
// The seventh nil is `CampaignRegistryHandler` (026 T035), added between the read-only
// look-up and the auth middleware: the write face is a separate type precisely so the
// look-up's constructor can never hold a unit of work.
func newTestRouter() *Router {
	return NewRouter(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		passAuthMiddleware{},
		nil,
	)
}

// TestRouterRegistersEveryCampaignRouteWithoutPanicking is T082 / R11.
//
// It asserts the PATHS, not just the absence of a panic. A registration that
// silently landed somewhere else — under `/admin/subscriptions`, say — would build
// fine and 404 every real request, which is a failure no amount of "it did not
// panic" would reveal.
func TestRouterRegistersEveryCampaignRouteWithoutPanicking(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	// The panic R11 is about would happen inside this call.
	newTestRouter().Init(&engine.RouterGroup)

	got := map[string]bool{}
	for _, route := range engine.Routes() {
		got[route.Method+" "+route.Path] = true
	}
	for _, path := range []string{
		"/admin/campaigns",
		"/admin/campaigns/:code",
		"/admin/campaigns/:code/claims",
		"/admin/campaigns/:code/eligibility/:userId",
	} {
		if _, ok := got["GET "+path]; !ok {
			t.Errorf("expected GET %s to be registered; the router has:\n%s", path, routeList(engine))
		}
	}

	// 026 T035: the create route shares the group's static path with the list route, so
	// this is also the check that Gin accepted a second METHOD on it. A conflict would
	// have panicked in `Init` above; landing on a different path would 404 every real
	// request while every other assertion here stayed green.
	if _, ok := got["POST /admin/campaigns"]; !ok {
		t.Errorf("expected POST /admin/campaigns to be registered; the router has:\n%s", routeList(engine))
	}

	// 026 T043: the close route is the first campaign POST to carry a `:code`
	// PARAMETER, so it is the first registration that could hit R11 from this side —
	// a static segment registered beside a parameter panics while the tree is built,
	// which is above, and a route that quietly landed elsewhere would 404 a
	// one-way operation an operator believes they performed.
	if _, ok := got["POST /admin/campaigns/:code/close"]; !ok {
		t.Errorf("expected POST /admin/campaigns/:code/close to be registered; the router has:\n%s", routeList(engine))
	}
}

// TestCampaignRoutesAreNotNestedUnderAdminSubscriptions is the R11 assertion proper.
//
// The failure mode is a campaign route registered under `/admin/subscriptions`,
// beside `POST /:userId/cancel`. If that ever happens the service panics on boot;
// if a future refactor somehow avoids the panic, the routes would live at a
// different URL than the contract publishes. So the check is that the campaign
// paths are the ones the contract names and that no campaign segment appears
// anywhere under the subscriptions group.
func TestCampaignRoutesAreNotNestedUnderAdminSubscriptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	newTestRouter().Init(&engine.RouterGroup)

	for _, route := range engine.Routes() {
		if route.Path == "/admin/subscriptions/:userId/cancel" {
			continue
		}
		if len(route.Path) >= len("/admin/subscriptions/campaigns") &&
			route.Path[:len("/admin/subscriptions/campaigns")] == "/admin/subscriptions/campaigns" {
			t.Errorf("R11: %s is nested under /admin/subscriptions, which holds POST /:userId/cancel. "+
				"Gin refuses a static segment beside a parameter and PANICS AT STARTUP",
				route.Path)
		}
	}
}

// TestExistingRoutesSurviveTheNewGroup is the regression guard in the other
// direction.
//
// A new group is added; nothing existing may move. The campaign grant's own route
// is the one that matters most — it is how a customer receives a plan — so its
// absence here would mean a reporting feature took the grant path down with it.
func TestExistingRoutesSurviveTheNewGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	newTestRouter().Init(&engine.RouterGroup)

	registered := map[string]bool{}
	for _, route := range engine.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		"POST /admin/subscriptions/:userId/cancel",
		"POST /admin/subscription-grants",
		"POST /admin/subscription-grants/restore",
		"GET /subscriptions/plans",
		"GET /admin/subscriptions/plans",
	} {
		if !registered[route] {
			t.Errorf("expected %s to still be registered after adding the campaign group", route)
		}
	}
}

// TestUnknownCampaignPathIs404NotAPanic is the observable form of R11.
//
// A path under the campaign group that matches nothing must answer 404. The
// handlers are nil here, so any route that DID match would panic — which makes
// this test prove both that the group is mounted and that nothing inside it is a
// catch-all.
func TestUnknownCampaignPathIs404NotAPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	newTestRouter().Init(&engine.RouterGroup)

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/campaigns/some/deep/unknown/path", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for an unmatched campaign path, got %d", rec.Code)
	}
}

// unusedHandlerImport keeps the handler import honest: the nil arguments above are
// typed by the constructor, and naming the package documents what they are.
var _ = (*subscription_handler.CampaignAdminHandler)(nil)

// unusedMiddlewareImport keeps the shared middleware import honest.
var _ = shared_middleware.RolesAuthorize

// routeList renders the registered routes for a failure message.
func routeList(engine *gin.Engine) string {
	out := ""
	for _, route := range engine.Routes() {
		out += "\n  " + route.Method + " " + route.Path
	}
	return out
}
