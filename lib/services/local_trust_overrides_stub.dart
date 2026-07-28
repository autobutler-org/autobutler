// Stub for non-io platforms (web). The browser owns TLS trust, and there is no
// HttpClient to override, so this is a no-op.
void installLocalTrustHttpOverrides() {}
