package commonuser

type NativeAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DeviceInformation
}

type NativeAuthByEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceInformation
}

type NativeRegistrationRequest struct {
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
