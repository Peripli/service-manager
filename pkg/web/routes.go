package web

const (
	apiVersion = "v1"

	// BrokersURL is the URL path to manage service brokers
	BrokersURL = "/" + apiVersion + "/service_brokers"

	// ServiceOfferingsURL is the URL path to manage service offerings
	ServiceOfferingsURL = "/" + apiVersion + "/service_offerings"

	// ServicePlansURL is the URL path to manage service plans
	ServicePlansURL = "/" + apiVersion + "/service_plans"

	// VisibilitiesURL is the URL path to manage visibilities
	VisibilitiesURL = "/" + apiVersion + "/visibilities"

	// PlatformsURL is the URL path to manage platforms
	PlatformsURL = "/" + apiVersion + "/platforms"

	// OSBURL is the OSB API base URL path
	OSBURL = "/" + apiVersion + "/osb"

	// MonitorHealthURL is the path of the healthcheck endpoint
	MonitorHealthURL = "/" + apiVersion + "/monitor/health"

	// InfoURL is the path of the info endpoint
	InfoURL = "/" + apiVersion + "/info"
)
