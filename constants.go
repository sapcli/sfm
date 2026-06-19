package sfm

const (
	URLLaunchpad        = "https://launchpad.support.sap.com"
	URLAccount          = "https://accounts.sap.com"
	URLAccountCoreAPI   = "https://core-api.account.sap.com/uid-core"
	URLAccountCDCApi    = "https://cdc-api.account.sap.com"
	URLAccountSSOIDP    = "https://cdc-api.account.sap.com/saml/v2.0/{k}/idp/sso/continue"
	URLAccountSAMLProxy = "https://account.sap.com/core/SAMLProxyPage.html"
	URLAccountAttrs     = "https://launchpad.support.sap.com/services/account/attributes"
	URLUserAdmin        = "https://launchpad.support.sap.com/services/odata/useradminsrv"
	URLSupportPortal    = "https://hana.ondemand.com/supportportal"
	URLGigyaJS          = "https://cdns.gigya.com/js/gigya.js"
	URLLearningHub      = "https://saplearninghub.plateau.com"
	URLPartnerEdge      = "https://partnermanagemyusers.cfapps.eu10-004.hana.ondemand.com"
)

const UserAgentChrome = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

const DefaultHTTPRetryAttempts = 3

var retryStatusForcelist = map[int]struct{}{
	413: {},
	429: {},
	500: {},
	502: {},
	503: {},
	504: {},
	509: {},
}
