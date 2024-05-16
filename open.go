package volcano

const (
	_Host   = "open.volcengineapi.com"
	_Scheme = "https"
)

type Config struct {
	AccessKey string `json:"access_key" yaml:"access_key"`
	SecretKey string `json:"secret_key" yaml:"secret_key"`
}

type OpenApi struct {
	Credentials
}
