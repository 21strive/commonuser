package commonuser

type NativeAuthRequestBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DeviceInformation
}

type NativeRegistrationRequestBody struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	DeviceInformation
}

type DeviceInformation struct {
	DeviceId   string `json:"deviceId"`
	DeviceInfo string `json:"deviceInfo"`
	UserAgent  string `json:"userAgent"`
}

type PatchRequestBody struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}
