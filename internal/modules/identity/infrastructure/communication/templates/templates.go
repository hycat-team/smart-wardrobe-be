package templates

import _ "embed"

//go:embed RegistrationOtp.html
var RegistrationOtpTemplate string

//go:embed ForgotPasswordOtp.html
var ForgotPasswordTemplate string
