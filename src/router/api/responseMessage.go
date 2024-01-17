package api

type SuccessResponse struct {
    Success bool            `json:"success"`
    Response any            `json:"response"`
}

type ErrorResponse struct {
    Success bool            `json:"success"`
    ErrorCode int           `json:"errorCode"`
    ErrorBody interface{}   `json:"errorBody"`
}
