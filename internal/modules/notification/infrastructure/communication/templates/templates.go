package templates

import _ "embed"

//go:embed RegistrationOtp.html
var OtpTemplate string

//go:embed ForgotPasswordOtp.html
var ForgotPasswordTemplate string

//go:embed PaymentSuccess.html
var PaymentSuccessTemplate string

//go:embed TopupSuccess.html
var TopupSuccessTemplate string

//go:embed RenewalSuccess.html
var RenewalSuccessTemplate string

//go:embed RenewalFailed.html
var RenewalFailedTemplate string

//go:embed WalletWarning.html
var WalletWarningTemplate string
