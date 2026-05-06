package response

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

type Message struct {
	Message string `json:"message"`
}
