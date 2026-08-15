# Project Complexity And Assurance Levels

Use the level to control proactive questioning depth, never to restrict what
the developer may document. A developer may request any advanced section or
artifact at any level.

## Classification Inputs

Assess available evidence for:

- Consequences of failure and reversibility.
- Data sensitivity and privacy.
- Money movement, regulation, legal obligations, health, or safety.
- Users, roles, organizations, and tenancy.
- External integrations and compatibility contracts.
- Availability, recovery, and operational expectations.
- Team size, expected lifetime, and change frequency.

Recommend a level from the strongest relevant signals and explain the reasons.
The developer selects the final level.

## Levels

| Level | Typical Profile | Proactive Depth |
|---|---|---|
| `L1 Personal` | Local or personal project; low-impact, reversible failures; no sensitive data | Establish purpose, primary journey, technical baseline, and basic validation |
| `L2 Product` | Team-maintained product; multiple users, persistence, integrations, or regular releases | Cover alternate behavior, contracts, CI, dependency policy, and experience |
| `L3 Business-Critical` | Failure affects customers, operations, money, or sensitive data | Cover compatibility, migrations, security, observability, resilience, and recovery |
| `L4 High-Assurance` | Regulatory, financial, legal, health, safety, or severe availability impact | Require explicit evidence for critical claims, auditability, threat analysis, disaster recovery, and independent verification |

## Initialization Modes

### New

Start from developer intent. Recommend technical defaults and ask only about
choices that materially affect product behavior, maintenance, or assurance.
Do not present an unselected technology as established.

### Existing

Inspect before interviewing:

- Manifests, version files, and lockfiles.
- Source and test structure.
- CI, build, and validation commands.
- Configuration and public interfaces.
- Schemas, migrations, and persistence code.
- Existing documentation and integration boundaries.

Treat repository facts as evidence, not questions. Ask about contradictions,
missing business intent, and consequential choices only.

### Hybrid

For a new slice in an existing project, inherit confirmed project conventions
and focus discovery on the new context, journeys, contracts, and deviations.

## Level Changes

Increase the recommended level when new evidence raises consequences or
assurance needs. Do not silently lower a developer-selected level. A level
change adjusts the remaining question frontier; it does not delete information
already documented.
