package web

const (
	apiVersion = "v1"

	// BrokersURL is the URL path to manage service brokers
	BrokersURL = "/" + apiVersion + "/service_brokers"

	// PlatformsURL is the URL path to manage platforms
	PlatformsURL = "/" + apiVersion + "/platforms"

	// SMCatalogURL is the URL path to the aggregated catalog
	SMCatalogURL = "/" + apiVersion + "/sm_catalog"

	// OSBURL is the OSB API base URL path
	OSBURL = "/" + apiVersion + "/osb"
)
