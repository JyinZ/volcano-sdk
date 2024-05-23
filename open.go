package volcano

import "net/url"

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

func (c *OpenApi) BuildUrl(path, version, action string) url.URL {
	return url.URL{
		Scheme: _Scheme,
		Host:   _Host,
		Path:   path,
		RawQuery: url.Values{
			"Version": []string{version},
			"Action":  []string{action},
		}.Encode(),
	}
}
