package profiles

import dash0 "github.com/dash0hq/dash0-api-client-go"

// ClientOptions returns [dash0.ClientOption] values that configure a client
// from this Configuration.
// Non-empty fields are mapped as follows:
//   - ApiUrl    -> [dash0.WithApiUrl]
//   - AuthToken -> [dash0.WithAuthToken]
//   - OtlpUrl   -> [dash0.WithOtlpEndpoint] with [dash0.OtlpEncodingJson]
//
// The Dataset field is not mapped because it is a per-request parameter, not a
// client-level setting.
// Use [Configuration.DatasetPtr] to convert it for API calls.
//
// Callers can append additional options to override or supplement the returned
// slice:
//
//	opts := cfg.ClientOptions()
//	opts = append(opts, dash0.WithUserAgent("my-tool/1.0"))
//	client, err := dash0.NewClient(opts...)
func (cfg *Configuration) ClientOptions() []dash0.ClientOption {
	var opts []dash0.ClientOption
	if cfg.ApiUrl != "" {
		opts = append(opts, dash0.WithApiUrl(cfg.ApiUrl))
	}
	if cfg.AuthToken != "" {
		opts = append(opts, dash0.WithAuthToken(cfg.AuthToken))
	}
	if cfg.OtlpUrl != "" {
		opts = append(opts, dash0.WithOtlpEndpoint(dash0.OtlpEncodingJson, cfg.OtlpUrl))
	}
	return opts
}
