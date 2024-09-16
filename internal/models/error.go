package models

type ErrorResponse struct {
	Err string `json:"reason"`
}

func ErrorResp(err string) *ErrorResponse {
	return &ErrorResponse{Err: err}
}

type Error struct {
	UserCaused bool
	desc       string
}

func NewParseError(desc string, userCaused ...bool) *Error {
	if len(userCaused) > 1 {
		panic("no more than 1")
	}

	user := false
	if len(userCaused) == 1 && userCaused[0] {
		user = true
	}

	return &Error{desc: desc, UserCaused: user}
}

func (e *Error) Response() *ErrorResponse {
	return &ErrorResponse{Err: e.desc}
}

func (e *Error) Error() string {
	return "parsing error"
}
