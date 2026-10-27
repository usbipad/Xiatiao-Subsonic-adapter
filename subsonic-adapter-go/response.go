package main

// Subsonic 协议版本与错误码，对应 Node 版 response.ts。
const subsonicVersion = "1.16.1"

const (
	ErrGeneric             = 0
	ErrMissingParameter    = 10
	ErrClientVersionTooLow = 20
	ErrServerVersionTooLow = 30
	ErrAuthentication      = 40
	ErrAuthorization       = 50
	ErrNotFound            = 70
)

// SubsonicError Subsonic 错误体。
type SubsonicError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// EnvelopeOptions 响应外壳选项。
type EnvelopeOptions struct {
	Type          string
	ServerVersion string
}

// OkResponse 构造成功响应。
func OkResponse(opts EnvelopeOptions, payload map[string]any) map[string]any {
	body := map[string]any{
		"status":        "ok",
		"version":       subsonicVersion,
		"type":          opts.Type,
		"serverVersion": opts.ServerVersion,
		"openSubsonic":  true,
	}
	for k, v := range payload {
		body[k] = v
	}
	return map[string]any{"subsonic-response": body}
}

// FailedResponse 构造失败响应。
func FailedResponse(opts EnvelopeOptions, code int, message string) map[string]any {
	return map[string]any{
		"subsonic-response": map[string]any{
			"status":        "failed",
			"version":       subsonicVersion,
			"type":          opts.Type,
			"serverVersion": opts.ServerVersion,
			"openSubsonic":  true,
			"error": SubsonicError{
				Code:    code,
				Message: message,
			},
		},
	}
}

// hasAuthParams 认证只校验参数形态，不校验口令内容（与 Node 版一致）。
func hasAuthParams(q map[string][]string) bool {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	has := func(k string) bool { return get(k) != "" }
	u := get("u") != ""
	token := has("t") && has("s")
	plain := has("p")
	return u && (token || plain)
}
