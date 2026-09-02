# Origin

TrustSentinel continuation of the **marshmallows / Meshmellows** project (2019) —
a lightweight secure-mesh networking platform for IoT: an encrypted peer overlay
with 2FA/U2F, built on the Noise Protocol (Go agent + broker + web). Conceptually
close to a self-hosted Tailscale-style secure overlay.

Recognition: INCIBE National Cybersecurity Competition 2019 (4th place) —
"lightweight cryptography implicit in IoT mesh architectures".

Imported clean-start (no upstream history), 2026-09-02, and sanitized (removed the
hackathon presentation; replaced a hardcoded frontend token with a placeholder).
gitleaks scan: clean. Its Go Noise agent also seeds trustsentinel/stk.
