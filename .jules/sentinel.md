## 2023-10-12 - Header Injection via Environment Variables
**Vulnerability:** HTTP Response Splitting / Header Injection
**Learning:** `responder.Spec.Validate()` correctly prevents control characters in header values designed by the user. However, when these headers contain environment variable placeholders (`${VAR}`), `responder.Render()` expanded them *after* validation. An attacker could potentially inject arbitrary headers or split the response by manipulating an allowed environment variable to contain `\r\n`.
**Prevention:** Sanitized the expanded environment variable values in `Render()` by stripping `\r` and `\n` characters before adding them to the final response headers.
