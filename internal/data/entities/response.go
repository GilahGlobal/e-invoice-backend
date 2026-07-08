package entities

type Response struct {
	Status     string      `json:"status,omitempty"`
	StatusCode int         `json:"status_code,omitempty"`
	Name       string      `json:"name,omitempty"`
	Message    string      `json:"message,omitempty"`
	Error      interface{} `json:"error,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Pagination interface{} `json:"pagination,omitempty"`
	Extra      interface{} `json:"extra,omitempty"`
}
