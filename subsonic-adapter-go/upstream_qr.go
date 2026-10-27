package main

// 上游扫码登录相关调用，供管理页使用。

// GetQrKey 获取二维码 key。channel 为空时使用上游默认通道（qq）。
// 上游支持的通道：qq / wechat / qqmusic。
func (u *QQUpstream) GetQrKey(channel string) (string, bool) {
	params := map[string]string{}
	if channel != "" {
		params["channel"] = channel
	}
	data := u.Get("/login/qr/key", params)
	if data == nil {
		return "", false
	}
	key := firstString(dict(data)["data"].(map[string]any)["unikey"])
	if key == "" {
		return "", false
	}
	return key, true
}

// GetQrImage 根据 key 生成二维码图片（data URL）。
func (u *QQUpstream) GetQrImage(key string) (string, bool) {
	data := u.Get("/login/qr/create", map[string]string{"key": key})
	if data == nil {
		return "", false
	}
	inner, _ := dict(data)["data"].(map[string]any)
	img := firstString(inner["qrimg"])
	if img == "" {
		return "", false
	}
	return img, true
}

// CheckQr 轮询扫码状态，返回 code 与会话 token。
func (u *QQUpstream) CheckQr(key string) (int, string) {
	data := u.Get("/login/qr/check", map[string]string{"key": key})
	if data == nil {
		return 0, ""
	}
	code := 0
	if n, ok := num(dict(data)["code"]); ok {
		code = n
	}
	token := ""
	if code == 803 {
		if c, ok := dict(data)["cookie"].(string); ok {
			token = qrCookieTokenOf(c)
		}
	}
	return code, token
}

// qrCookieTokenOf 从 Set-Cookie 字符串中取 qqmusic_session。
func qrCookieTokenOf(cookie string) string {
	for _, entry := range splitSemicolon(cookie) {
		idx := indexByte(entry, '=')
		if idx <= 0 {
			continue
		}
		if trimSpace(entry[:idx]) == "qqmusic_session" {
			return trimSpace(entry[idx+1:])
		}
	}
	return ""
}
