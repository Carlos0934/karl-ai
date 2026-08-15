# System Documentation Model

This model separates business context, observable journeys, and the current
technical baseline. It is independent of whether the source is a conversation,
issue, bug report, pull request, or direct request.

## Quick Map

| Document | Answers | Excludes |
|---|---|---|
| `CONTEXT.md` | What business context exists? | Technology and UI/UX |
| `JOURNEY.md` | What behavior and experience must occur? | Internal implementation |
| `DESIGN.md` | How is the project technically organized today? | Change plans and history |

## Foundation Metadata

Keep project-wide analysis state in the frontmatter of `docs/CONTEXT.md`. Do
not duplicate it in DESIGN.md or journey files.

| Field | Allowed Values Or Format |
|---|---|
| `foundation_state` | `draft`, `established`, `needs-review` |
| `project_state` | `unknown`, `discovery`, `development`, `maintenance`, `modernization` |
| `analysis_state` | `pending`, `in-progress`, `complete`, `stale` |
| `project_mode` | `unknown`, `new`, `existing`, `hybrid` |
| `assurance_level` | `pending`, `L1`, `L2`, `L3`, `L4` |
| `last_analyzed_at` | ISO 8601 timestamp |
| `analysis_summary` | One-line evidence-based summary |

Set `foundation_state` to `established` only when required context and design
content is coherent at the selected assurance level. Use `needs-review` when
repository evidence contradicts the baseline or important unknowns remain.
Mark `analysis_state` as `stale` when meaningful code, contracts, architecture,
or business behavior changed without a foundation refresh.

## Recommended Structure

```text
docs/
|-- CONTEXT.md
|-- DESIGN.md
|-- journeys/
|   `-- <journey-name>/
|       `-- JOURNEY.md
|-- architecture/
|   |-- system-architecture.mmd
|   |-- runtime-flow.mmd
|   `-- dependencies.mmd
|-- data-models/
|   |-- domain-model.md
|   |-- persistence.dbml
|   `-- schemas/
|-- contracts/
|   |-- http/
|   |-- cli/
|   |-- events/
|   `-- library/
`-- uiux/
    |-- screens/
    |-- wireframes/
    `-- interaction-flows/
```

Create optional folders only when the project needs them. Do not add a
`context/` folder for one root context. Split contexts only after the project
has independently meaningful business slices.

## CONTEXT.md: Business Baseline

CONTEXT.md defines a bounded part of the business or domain:

- Identity, purpose, and scope.
- Human and non-human actors.
- External systems used by the context.
- Canonical concepts and language.
- Capabilities and journey map.
- Business rules and invariants.
- Significant outcomes and side effects.

It must not describe technologies, dependencies, databases, queues, project
structure, technical validation, or visual experience.

### Actors And External Systems

An actor initiates, participates in, or receives a business outcome. A
non-human API consumer can therefore be an actor.

An external system is used by this context to complete work. It is not a
business actor merely because it has an API.

```text
Does the element initiate, participate in, or receive a business outcome?
    Yes -> Actor
    No, the context uses it to perform work -> External System
```

### Canonical Language

Keep canonical terms directly in CONTEXT.md; do not create a separate glossary
by default. Add a term when it is stable and relevant across journeys.

## JOURNEY.md: Behavior And Experience

JOURNEY.md defines one complete business behavior. A journey may involve
multiple actors, interfaces, state changes, alternate paths, failure paths, and
side effects. It is not required to be atomic.

Every significant journey should identify:

- Purpose and trigger.
- Initiating, participating, and receiving actors.
- Business context, inputs, and preconditions.
- Observable actor interactions.
- Main, alternate, and failure flows.
- Rules, state changes, side effects, and reversibility.
- Expected outcomes, postconditions, and acceptance criteria.

### Interaction Surfaces

An interaction surface is an observable boundary contract:

```text
Actor + Surface + Trigger + Input + Response + Business Result
```

Use only the surfaces relevant to the journey:

| Surface | Useful Detail |
|---|---|
| HTTP | Method, endpoint, request, responses, errors |
| CLI | Command, options, output, errors, exit codes |
| ASCII UI | Entry point, actions, visible state, feedback |
| Other | Channel, input, response, business result |

Do not include classes, private methods, database queries, queues, frameworks,
or internal protocols.

### Experience Details

When experience matters, document it in the journey:

- Experience goal and entry point.
- What the actor perceives and can do.
- Initial, processing, empty, validation, success, and failure states.
- Feedback, accessibility requirements, and visual references.

Keep business state distinct from interface state. For example,
`Reservation Confirmed` is a business state; `Confirmation Screen Displayed`
is an interface state.

## DESIGN.md: Current Technical Baseline

DESIGN.md describes the project as it is intended to work now. Maintain it in
place. It is not a historical decision log and must not contain implementation
steps for one change.

It covers:

1. Design scope and architecture goals.
2. Architecture and system organization.
3. Technology stack, tooling, and version policies.
4. Project structure and development conventions.
5. Domain and persistence models.
6. Contracts and external interfaces.
7. Testing methodology and code validation.
8. Optional UI/UX baseline.
9. Operational, security, and quality requirements.

### Architecture

Describe the current architecture style, major technical areas, dependency
direction, runtime shape, and external integrations. Put detailed Mermaid views
in `architecture/` and link them from DESIGN.md.

### Technology And Tooling

Record runtime versions, major dependencies, development tools, lockfiles,
version sources, compatibility policy, and reproducibility requirements. This
is a usable technical baseline, not an exhaustive package inventory.

### Project Structure

Describe the current directory tree and the responsibility of each important
path. State both what belongs and what must not be placed there.

### Data Models

Keep domain and persistence representations distinct:

```text
Domain model
    Entities, value objects, aggregates, invariants
        |
        +--> Persistence model
                Tables, fields, indexes, and relationships
```

Use DBML for relational persistence when appropriate. Use JSON Schema or
another suitable representation for document validation. Do not force these
formats onto applications that do not need them.

### Contracts

Choose the contract format according to the application:

| Interface | Suitable Formats |
|---|---|
| HTTP | OpenAPI, JSON Schema, documented HTTP contract |
| CLI | Markdown command, input, output, error, and exit-code contract |
| Events | AsyncAPI, JSON Schema, documented event contract |
| Library | Public API and usage contract |

OpenAPI can describe complete HTTP semantics. JSON Schema can define payloads
but does not by itself describe the full endpoint behavior.

### Testing And Validation

Document the project methodology, not only tool names. State:

- Which test levels exist and what each level proves.
- Where tests, fixtures, and test data live.
- The validation sequence.
- Local and CI commands.
- Required quality gates.
- Which journeys require contract or end-to-end coverage.

### UI/UX Baseline

Use this optional section for project-wide experience rules: visual language,
navigation, shared interaction patterns, required UI states, accessibility,
responsive behavior, and links to detailed artifacts under `uiux/`.

Project-level principles belong in DESIGN.md. Journey-specific experience
belongs in the relevant JOURNEY.md.

## Relationship Between Documents

```text
CONTEXT.md
    Defines business scope, concepts, and rules
        |
        +--> JOURNEY.md
        |       Defines observable behavior and experience
        |
        +--> DESIGN.md
                Defines the current technical baseline
                    |
                    +--> architecture/
                    +--> data-models/
                    +--> contracts/
                    +--> uiux/
```

DESIGN.md is a hub for detailed technical artifacts. Link instead of copying
large diagrams, schemas, contracts, or models into the main document.

## Authoring Order

1. Establish CONTEXT.md and its business boundary.
2. Define relevant journeys and link them from the context.
3. Capture the technical baseline in DESIGN.md.
4. Add architecture, model, contract, and UI/UX artifacts only as needed.
5. Create work-specific plans only for concrete changes.

## Duplication Rules

- Keep domain definitions and cross-journey rules in CONTEXT.md.
- Keep complete observable behavior in JOURNEY.md.
- Keep current technical rules in DESIGN.md.
- Keep detailed schemas and diagrams in their specialized folders.
- Use summaries and links when a document needs another level of detail.

## Completion Criteria

The baseline is sufficiently documented when:

- The business boundary is explicit.
- Actors and external systems are distinguished.
- Canonical terms and journeys are discoverable.
- Journeys identify interactions, outcomes, state changes, and side effects.
- Relevant experience states are documented.
- Architecture, project structure, and technology versions are clear.
- Data models and public contracts use suitable representations.
- Testing and validation commands are reproducible.
- Technical history and change-specific plans are kept out of the baseline.
