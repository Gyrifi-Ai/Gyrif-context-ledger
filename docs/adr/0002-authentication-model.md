# ADR 0002: Trusted deployment boundary without application authentication

- Status: Accepted
- Date: 2026-09-01

## Context

Gyrifi is a local-first context ledger intended to run as a single instance inside a company's controlled VM or private VPC. GRF-220 proposed application-managed operator passwords, browser sessions, and Ledger-scoped ingestion tokens. That design would add credential lifecycle, login UI, session state, secret rotation, and authorization policy to a product whose deployment boundary already provides access control.

The product still needs an honest trust model. Without application authentication, every caller that can reach the application listener can submit Changes, evaluate and approve Proposals, release to the target, create rollback Proposals, and provide the recorded approval actor. A private network is therefore a security boundary, not merely a deployment convenience.

## Decision

Gyrifi will not implement application-level authentication or authorization in the current product.

- Deployments MUST restrict the application listener to a trusted host, VM, private VPC, VPN, service mesh, or authenticated reverse proxy.
- The public internet is not a supported deployment target.
- Operators are responsible for network admission, TLS termination when traffic leaves the host, and identity enforcement at any reverse proxy.
- Gyrifi continues to accept the approval `actor` in the request body. That value is an operator-provided audit attribution, not a cryptographically authenticated identity.
- Studio has no login screen, session cookie, token-management screen, or credential persistence.
- Automated ingestion uses the same private endpoint as Studio. Network policy or a reverse proxy may restrict producers more narrowly when required.
- `/healthz` and `/readyz` remain on the application listener. Prometheus metrics remain on the separately configured loopback-only listener.
- `GYRIFI_AUTH_DISABLED`, `GYRIFI_SESSION_SECRET`, operator records, ingestion-token records, and authentication CLI commands are not introduced.
- No authentication migration is created. Migration number `005` remains available to the next schema-changing ticket.

GRF-220 is closed as superseded by this decision, with its implementation acceptance criteria explicitly not implemented. This is an intentional product-scope decision, not a claim that private networks authenticate callers.

GRF-226 remains useful as availability protection against accidental or runaway trusted clients, but it no longer depends on an authenticated principal. It will key limits by validated client address, honoring forwarded addresses only from explicitly configured trusted proxies. Login-specific limits and per-principal keying are removed from that ticket.

## Operational requirements

A supported deployment must satisfy all of the following:

1. Port `8080` is not exposed to the public internet.
2. Firewall, security-group, VPC, VPN, service-mesh, or reverse-proxy policy admits only trusted company clients.
3. If requests cross a host boundary, TLS is terminated before they reach an untrusted network segment.
4. Reverse proxies remove untrusted forwarding headers and set their own canonical forwarding metadata.
5. Qdrant, llama.cpp, SQLite, the object store, and the metrics listener remain unreachable from untrusted networks.
6. Operators understand that the `actor` field is asserted by the caller and must not be treated as externally verified identity.

The shipping local Compose configuration continues to bind the host application port to `127.0.0.1`. A company deployment may bind to a private interface only when the surrounding network controls satisfy this ADR.

## Consequences

### Positive

- Gyrifi remains simple to deploy and operate as a local-first, single-instance system.
- No password database, bearer-token lifecycle, session signing key, login flow, or authentication dependency is added.
- Studio and ingestion clients require no credential bootstrap or secret distribution.
- The runtime remains usable behind company-standard identity-aware proxies without duplicating their identity systems.

### Trade-offs

- Any caller admitted by the deployment boundary has full governance authority.
- Gyrifi cannot independently distinguish an operator from an ingestion producer.
- Approval actor attribution is not authenticated and can be spoofed by an admitted caller.
- A firewall or private VPC alone does not protect against a compromised internal workload.
- Per-user revocation, least-privilege Ledger tokens, and application-level audit identity are unavailable.
- Documentation and status surfaces must never imply that Gyrifi itself authenticates requests.

## Rejected alternatives

- **Built-in passwords, signed sessions, and ingestion tokens:** rejected because the approved deployment model delegates identity and admission to company infrastructure.
- **Optional built-in authentication:** rejected for now because optional security paths double the test and operational surface and tend to be misconfigured. A future ticket may revisit this if deployments require it.
- **Trusting arbitrary `X-Forwarded-For`:** rejected because callers could forge identities used for rate-limit keys. Forwarded addresses are accepted only from configured trusted proxies.
- **Public deployment with no authentication:** rejected. This ADR authorizes only controlled VM/VPC/private-network operation.
- **Treating `actor` as verified identity:** rejected because no application credential binds the value to a person.

## Revisit triggers

A new ADR is required before adding any of the following:

- public-internet exposure;
- direct multi-tenant access;
- application-managed users, sessions, API tokens, or roles;
- cryptographically attributable approvals;
- per-Ledger producer isolation not supplied by network infrastructure;
- multiple replicas requiring shared identity or session state.
