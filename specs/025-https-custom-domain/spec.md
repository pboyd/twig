# Feature Specification: HTTPS with Custom Domain

**Feature Branch**: `025-https-custom-domain`

**Created**: 2026-05-31

**Status**: Draft

**Input**: User description: "This app needs to be served over HTTPS with a TLS certificate signed by a widely-recognized provider. The domain name will need to be customized for every deployment."

## Clarifications

### Session 2026-05-31

- Q: Is the certificate authority fixed or operator-configurable per deployment? → A: Fixed default public CA (e.g. Let's Encrypt, with automatic fallback); not operator-configurable.
- Q: Should the deployment enforce HSTS (Strict-Transport-Security)? → A: No HSTS; rely on the HTTP→HTTPS redirect only.
- Q: What is the minimum TLS protocol version? → A: TLS 1.3 minimum (reject TLS 1.2 and below).
- Q: How are public-CA issuance rate limits handled during repeated/non-production deployments? → A: Per-deployment toggle to use the CA's staging/test environment, defaulting to production.
- Q: Is the deployment reachable from the public internet, or on a private network? → A: Deployed on a LAN behind a private/internal IP and **not** reachable from the public internet, so challenge methods that require inbound public connectivity to the deployment itself cannot be used.
- Q: How is control of the domain proven for certificate issuance, given the deployment is not publicly reachable? → A: Through a DNS-based challenge — the system proves domain control by creating records in the domain's DNS, using operator-supplied DNS-provider credentials (the domain is hosted at Cloudflare in the reference deployment) — rather than by answering a challenge on the deployment's own address.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - End user reaches the app securely (Priority: P1)

A person opens the deployed application in a web browser using its domain name. The connection is encrypted and the browser shows the standard secure-connection indicator with no certificate warnings, because the certificate is signed by a certificate authority the browser already trusts.

**Why this priority**: This is the core outcome of the feature. Without a trusted, encrypted connection, end users either cannot reach the app safely or are confronted with scary browser warnings that erode trust and block adoption. Everything else exists to make this experience possible.

**Independent Test**: Deploy the app to a domain, then visit it from a clean browser and from an automated TLS checker. The connection succeeds over HTTPS, presents a certificate chaining to a publicly-trusted root, and produces no security warnings.

**Acceptance Scenarios**:

1. **Given** the app is deployed at its configured domain, **When** an end user visits the site over HTTPS, **Then** the browser establishes an encrypted connection and displays the secure indicator with no certificate warning.
2. **Given** an end user navigates to the app using a plain `http://` address, **When** the request reaches the server, **Then** they are automatically redirected to the secure `https://` equivalent of the same address.
3. **Given** the certificate has been issued, **When** its validity is inspected, **Then** it is signed by a widely-recognized (publicly-trusted) certificate authority and lists the deployment's configured domain.

---

### User Story 2 - Operator configures the domain for a deployment (Priority: P1)

An operator standing up a new deployment of the app supplies the domain name that this particular deployment should answer to. The deployment then serves the app—and obtains its certificate—for exactly that domain, without the operator hand-editing certificates or low-level server internals.

**Why this priority**: The feature explicitly requires the domain to be customizable per deployment. Two deployments (e.g., staging and production, or two separate customers) must be able to run from the same codebase on different domains. Without a single, clear configuration point, every deployment becomes a manual, error-prone fork.

**Independent Test**: Provision two deployments with two different configured domains. Each serves the app at its own domain with a valid certificate for that domain, and neither requires code changes—only the domain configuration value differs.

**Acceptance Scenarios**:

1. **Given** an operator is preparing a deployment, **When** they set the deployment's domain configuration value, **Then** the deployment serves the app at that domain and obtains a certificate for it.
2. **Given** two deployments configured with different domains, **When** each is brought up, **Then** each presents a valid certificate matching its own configured domain.
3. **Given** an operator changes a deployment's configured domain and redeploys, **When** the deployment comes back up, **Then** it serves the app and obtains a certificate for the new domain.

---

### User Story 3 - Certificate stays valid over time without manual work (Priority: P2)

A deployment runs unattended for months. Its certificate is renewed automatically before it expires, so end users never encounter an expired-certificate warning and the operator is not paged to manually rotate certificates.

**Why this priority**: Publicly-trusted certificates are short-lived by design, so a one-time issuance is not enough for a real deployment. Automatic renewal turns a manual recurring chore into a non-event. It is P2 because a deployment is functional at launch without it, but it is essential for unattended longevity.

**Independent Test**: Simulate or wait for a deployment to approach its certificate's expiry and confirm the certificate is renewed automatically and served without operator intervention or downtime.

**Acceptance Scenarios**:

1. **Given** a deployment with a valid certificate approaching expiry, **When** the renewal window is reached, **Then** the certificate is renewed automatically and the new certificate is served without manual action.
2. **Given** a renewal attempt fails transiently, **When** the system retries, **Then** it continues attempting renewal ahead of expiry rather than giving up after a single failure.

---

### Edge Cases

- **Clients cannot resolve the domain to the deployment**: To reach the app, LAN clients must resolve the configured domain to the deployment's (internal) address; if they cannot, they will not reach it even though the certificate is valid. (Certificate *issuance* itself does not require the domain to point at the deployment, because domain control is proven through DNS.) Such misresolution should be diagnosable rather than presenting as a silent failure.
- **Missing, invalid, or under-privileged DNS credentials**: If the DNS-provider credentials are absent, wrong, or lack permission to manage the configured domain's records, the DNS-based challenge cannot complete; the deployment should fail fast with a clear, actionable error rather than silently serving an untrusted or self-signed certificate.
- **DNS challenge-record propagation delay**: The DNS records created to satisfy the challenge may take time to become visible to the certificate authority; the system should accommodate propagation rather than failing on the first check.
- **Misconfigured or empty domain value**: If no domain is configured, or the value is not a valid hostname, the deployment should fail fast with a clear configuration error rather than starting in a broken state.
- **First request before the certificate is ready**: While a brand-new deployment is still obtaining its first certificate, requests should be handled gracefully (e.g., a brief delay or retry) rather than presenting an untrusted certificate.
- **Renewal outage**: If renewal repeatedly fails up to the point of expiry, the situation should be observable to the operator (logged/surfaced) so it can be addressed before users are affected.
- **Plain-HTTP access**: Requests arriving over unencrypted HTTP are redirected to HTTPS rather than served insecurely.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The application MUST be reachable by end users over an encrypted HTTPS connection.
- **FR-002**: In production mode, the TLS certificate presented to end users MUST be signed by a widely-recognized, publicly-trusted certificate authority so that mainstream browsers and clients trust it without manual configuration.
- **FR-003**: The certificate presented MUST match the deployment's configured domain name.
- **FR-004**: Operators MUST be able to set the domain name for each deployment through a single configuration value, without modifying application code.
- **FR-005**: Each deployment MUST be able to use a different domain than any other deployment built from the same codebase.
- **FR-006**: The system MUST automatically obtain the certificate for the configured domain as part of bringing the deployment up, without the operator manually generating or installing certificate files, and without depending on the deployment being reachable from the public internet.
- **FR-007**: The system MUST automatically renew the certificate before it expires, without operator intervention.
- **FR-008**: Requests arriving over plain HTTP MUST be redirected to the HTTPS equivalent of the same address.
- **FR-009**: If the configured domain is missing or invalid, the deployment MUST fail with a clear, actionable error rather than starting in a broken or insecure state.
- **FR-010**: If a certificate cannot be obtained or renewed, the condition MUST be observable to the operator (e.g., surfaced in logs or status) rather than failing silently.
- **FR-011**: In production mode, the system MUST NOT present an untrusted or self-signed certificate to end users in normal operation.
- **FR-012**: The system MUST negotiate connections using TLS 1.3 as the minimum protocol version and MUST reject connection attempts using TLS 1.2 or any earlier protocol version.
- **FR-013**: The deployment MUST provide a per-deployment toggle selecting the certificate authority's environment (production vs. staging/test), defaulting to production. The staging/test environment is intended only for non-production bring-ups to avoid production issuance rate limits; certificates issued from it are not expected to be publicly trusted, so the trusted-certificate requirements (FR-002, FR-011) apply to production mode.
- **FR-014**: Certificate issuance and renewal MUST NOT require the deployment to be reachable from the public internet. The system MUST be able to obtain and renew certificates for a deployment that runs on a private network (e.g., a LAN behind a private/internal IP address).
- **FR-015**: The system MUST prove control of the configured domain by way of that domain's DNS (a DNS-based challenge), rather than by answering a challenge directed at the deployment's own network address.
- **FR-016**: Operators MUST be able to supply, as per-deployment configuration, the DNS-provider credentials needed for the system to automatically create and remove the DNS records the challenge requires. These credentials MUST be handled as secrets and MUST NOT be exposed in logs, status output, or other operator-visible surfaces.
- **FR-017**: If the DNS-provider credentials are missing, invalid, or lack permission to manage the configured domain's records, the deployment MUST fail with a clear, actionable error (consistent with FR-009 and FR-010) rather than starting in a broken or insecure state.

### Key Entities *(include if feature involves data)*

- **Deployment domain configuration**: The single per-deployment setting that names the domain this deployment serves. It is the input that drives both which address the app answers on and which domain the certificate is issued for.
- **CA environment toggle**: A per-deployment setting selecting the certificate authority's production or staging/test environment. Defaults to production; staging is used only for non-production bring-ups to avoid issuance rate limits.
- **DNS-provider credentials**: A per-deployment secret granting the system permission to create and remove records in the configured domain's DNS so it can complete the DNS-based domain-control challenge automatically. Scoped to the domain in question; treated as a secret and never surfaced in operator-visible output.
- **TLS certificate**: The credential presented to end users that proves the server's identity and enables encryption. Its key attributes are the domain(s) it covers, the issuing certificate authority (which must be publicly trusted), and its validity period.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Visiting a deployment's configured domain in a current mainstream browser results in a secure connection with zero certificate warnings, 100% of the time.
- **SC-002**: A new deployment can be configured for a new domain by changing a single configuration value and requires no application code changes to serve that domain securely.
- **SC-003**: A fresh deployment reaches a working HTTPS state automatically within 10 minutes of being brought up (allowing for DNS challenge-record propagation), with no manual certificate steps and without requiring inbound public connectivity to the deployment.
- **SC-004**: Certificates are renewed automatically ahead of expiry such that, across the deployment's lifetime, end users encounter zero expired-certificate warnings.
- **SC-005**: An automated external TLS check confirms the presented certificate chains to a publicly-trusted root and matches the configured domain.
- **SC-006**: 100% of plain-HTTP requests to a deployment are redirected to HTTPS.
- **SC-007**: An automated TLS check confirms the deployment negotiates TLS 1.3 and refuses handshakes offering only TLS 1.2 or earlier.
- **SC-008**: A deployment running on a private/internal IP with no inbound connectivity from the public internet still obtains a publicly-trusted certificate for its configured domain, verified by an external TLS check of that certificate.

## Assumptions

- **Reverse-proxy / edge termination**: The deployment already fronts the application with an edge layer capable of terminating TLS (consistent with the existing deployment architecture). This feature governs how that layer is configured for HTTPS and the per-deployment domain, not the introduction of a brand-new serving component.
- **Automatic issuance from a public CA**: "Signed by a widely-recognized provider" is satisfied by automatic issuance from a free, publicly-trusted certificate authority (ACME-style, e.g., the kind used for automatic HTTPS). Purchasing certificates from a specific commercial vendor is not required. Per clarification, the CA is a fixed default (e.g. Let's Encrypt, with the edge layer's automatic fallback) and is **not** an operator-configurable per-deployment value.
- **Private-network deployment**: The deployment runs on a LAN behind a private/internal IP and is **not** reachable from the public internet. It serves end users over HTTPS on the standard port (443) and redirects plain HTTP (80) for LAN clients; because issuance uses a DNS-based challenge, inbound public connectivity to the deployment on ports 80/443 is **not** required to obtain or renew certificates.
- **Operator controls the domain's DNS via a supported provider**: The configured domain is hosted at a DNS provider (Cloudflare in the reference deployment) that supports automated, credential-driven record management. The operator supplies credentials for that provider so the system can create and remove the records the challenge requires. Issuance therefore proves domain control through DNS rather than through the deployment answering a network challenge.
- **Name resolution on the LAN**: LAN clients can resolve the configured (publicly-registered) domain to the deployment's internal address (e.g., via internal/split-horizon DNS or equivalent). The certificate is still issued by a widely-recognized public CA, so those clients see no certificate warnings even though the host is on a private address.
- **Single domain per deployment**: Each deployment serves one configured domain. Serving multiple distinct domains or wildcard domains from a single deployment is out of scope for this feature.
- **Bring-your-own-certificate is out of scope**: Supplying a manually-purchased or internally-issued certificate file is not part of this feature; issuance is automatic.
- **HSTS out of scope**: Per clarification, the deployment does not emit a Strict-Transport-Security (HSTS) header; secure access is enforced via the HTTP→HTTPS redirect (FR-008) rather than HSTS. Enabling HSTS may be revisited later.
