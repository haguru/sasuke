package routes

var (
	SignupDurationSecondsBuckets = []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	LoginDurationSecondsBuckets  = []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10}
)

const (
	// API route constants
	CreateRouteAPI  = "/create"
	MetricsRouteAPI = "/metrics"
	LoginRouteAPI   = "/login"
	SignupRouteAPI  = "/signup"

	// Content-Type constants
	ContentType     = "Content-Type"
	ContentTypeJson = "application/json"

	// metrics constants
	SignupRequestsTotal       = "signup_requests_total"
	SignupRequestsTotalHelp   = "Total number of signup requests received"
	SignupSuccessTotal        = "signup_success_total"
	SignupSuccessTotalHelp    = "Total number of successful signup requests"
	SignupErrorsTotal         = "signup_errors_total"
	SignupErrorsTotalHelp     = "Total number of errors during signup requests"
	SignupDurationSeconds     = "signup_duration_seconds"
	SignupDurationSecondsHelp = "Duration of signup requests in seconds"
	LoginRequestsTotal        = "login_requests_total"
	LoginRequestsTotalHelp    = "Total number of login requests received"
	LoginSuccessTotal         = "login_success_total"
	LoginSuccessTotalHelp     = "Total number of successful login requests"
	LoginFailedTotal          = "login_failed_total"
	LoginFailedTotalHelp      = "Total number of failed login requests"
	LoginDurationSeconds      = "login_duration_seconds"
	LoginDurationSecondsHelp  = "Duration of login requests in seconds"
	LoginRateLimitedTotal     = "login_rate_limited_total"
	LoginRateLimitedTotalHelp = "Total number of login requests that were rate limited"
)
