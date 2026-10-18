## 2024-10-18 - [Security Discovery Journal]
**Vulnerability:** N/A yet, exploring
**Learning:** N/A yet
**Prevention:** N/A yet
## 2024-10-18 - CRLF Injection via Environment Variable Expansion
**Vulnerability:** HTTP Response splitting/CRLF injection is possible when an environment variable containing `\r\n` is used inside a response header via `${VAR}` expansion. The validation of `Spec.Headers` happens BEFORE environment variables are expanded, allowing the rendered response to have headers with CRLF, causing `http.ResponseWriter` to split the response.
**Learning:** Checking inputs early is good, but when inputs are dynamically evaluated (e.g. `EnvPolicy.Expand`), the expanded form must either be validated again or the expansion logic must sanitize its output. The system validated `s.Headers` in `Set()` but didn't catch CRLF introduced by `${VAR}` at `Render()` time.
**Prevention:** Either validate the expanded headers in `Render()` and strip/reject invalid characters, or sanitize the output of `EnvPolicy.Expand` when it's used in header values. We will sanitize the header value directly during expansion in `Render()` or `EnvPolicy.Expand` by replacing `\n` and `\r` with spaces, as Go's `http.ResponseWriter` handles CRLF in headers very poorly depending on how it's written.
