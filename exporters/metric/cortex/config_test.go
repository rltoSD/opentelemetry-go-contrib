package cortex

var validYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
`)

var noTimeoutYAML = []byte(`url: /api/prom/push
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
`)

var noURLYAML = []byte(`remote_timeout: 30s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
`)

var twoPasswordsYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
name: Valid Config Example
basic_auth:
  username: user
  password: password
  password_file: passwordfile
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
`)

var twoBearerTokensYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
bearer_token_file: bearertokenfile
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
`)
