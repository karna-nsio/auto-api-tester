package types

// Endpoint represents an API endpoint with its parameters and test data
type Endpoint struct {
	Method     string
	Path       string
	Parameters []Parameter
	TestData   EndpointTestData
	Responses  map[int]Response
}

// EndpointTestData represents test data for a specific endpoint
type EndpointTestData struct {
	PathParams  map[string]interface{} `json:"path_params,omitempty"`
	QueryParams map[string]interface{} `json:"query_params,omitempty"`
	Body        interface{}            `json:"body,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
}

// Parameter represents an API parameter
type Parameter struct {
	Name        string
	In          string
	Required    bool
	Schema      interface{}
	ContentType string
}

// Response represents an API response
type Response struct {
	Description string
	Schema      interface{}
}
