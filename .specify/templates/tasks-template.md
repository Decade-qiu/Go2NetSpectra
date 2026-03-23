---

description: "Task list template for Go2NetSpectra feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Verification**: Verification tasks are REQUIRED whenever the touched path
affects packet parsing, aggregation, contracts, config, query, alerting, AI,
or deployment behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Service entrypoints: `cmd/...`
- Core runtime logic: `internal/...`
- Public and derived contracts: `api/proto/v1/`, `api/gen/v1/`
- Reusable libraries: `pkg/...`
- Runtime config and deployment: `configs/`, `deployments/`
- Fixtures and smoke assets: `test/`, `scripts/`

## Phase 1: Setup (Shared Context)

**Purpose**: Align feature scope with the existing pipeline, contracts, and
runtime surface

- [ ] T001 Review `spec.md`, `plan.md`, and the constitution; list touched
          packages, binaries, contracts, configs, scripts, and deployment
          assets
- [ ] T002 [P] Add or refresh packet fixtures and helper inputs under `test/`
          or `scripts/`
- [ ] T003 [P] If contracts or config will change, stage regeneration and sync
          work for `api/gen/v1/`, `configs/`, and `deployments/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core changes that MUST be complete before any user story can be
implemented safely

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Update shared contracts or schemas in `api/proto/v1/` or
          `configs/config.yaml`
- [ ] T005 [P] Wire impacted service and package boundaries in `cmd/` and
          `internal/`
- [ ] T006 [P] Add or update factory, registration, query, or storage plumbing
          in `internal/factory/`, `internal/engine/manager/`, or
          `internal/query/`
- [ ] T007 Define shutdown, snapshot/reset, backpressure, and error-handling
          behavior for any new goroutines, tickers, or channels
- [ ] T008 Sync deployment/runtime defaults in `deployments/` and developer
          docs when runtime behavior changes

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Verification for User Story 1 ⚠️

> **NOTE: Add the strongest relevant verification for the touched path before
> or alongside implementation.**

- [ ] T009 [P] [US1] Add or extend tests in `internal/[area]/[name]_test.go`
- [ ] T010 [P] [US1] Add pcap/script/client smoke coverage in `test/[fixture]`
          or `scripts/[tool]/...`

### Implementation for User Story 1

- [ ] T011 [P] [US1] Implement core logic in `internal/[bounded-context]/...`
- [ ] T012 [US1] Wire the entrypoint, contract, or API surface in
          `cmd/[service]/main.go` or `api/proto/v1/[contract].proto`
- [ ] T013 [US1] Regenerate derived artifacts in `api/gen/v1/` and update
          dependent clients/servers
- [ ] T014 [US1] Update config, deployment, or docs for story 1 in `configs/`,
          `deployments/`, `README.md`, or `doc/`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Verification for User Story 2 ⚠️

- [ ] T015 [P] [US2] Add or extend tests in `internal/[area]/[name]_test.go`
- [ ] T016 [P] [US2] Add targeted smoke coverage in `test/[fixture]`,
          `scripts/`, or service clients

### Implementation for User Story 2

- [ ] T017 [P] [US2] Implement story-specific logic in
          `internal/[bounded-context]/...`
- [ ] T018 [US2] Update service wiring, query paths, or storage outputs in
          `cmd/`, `internal/query/`, or writer packages
- [ ] T019 [US2] Sync generated artifacts, config, and deployment changes in
          `api/gen/v1/`, `configs/`, and `deployments/`
- [ ] T020 [US2] Integrate with User Story 1 components while preserving
          independent testability

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Verification for User Story 3 ⚠️

- [ ] T021 [P] [US3] Add or extend tests in `internal/[area]/[name]_test.go`
- [ ] T022 [P] [US3] Add targeted smoke coverage in `test/[fixture]`,
          `scripts/`, or deployment validation

### Implementation for User Story 3

- [ ] T023 [P] [US3] Implement story-specific logic in
          `internal/[bounded-context]/...`
- [ ] T024 [US3] Update service wiring, scripts, or API surfaces in `cmd/`,
          `scripts/`, or `api/proto/v1/`
- [ ] T025 [US3] Sync generated artifacts, config, deployment, and docs in
          `api/gen/v1/`, `configs/`, `deployments/`, or `doc/`
- [ ] T026 [US3] Validate cross-story behavior without breaking existing
          query, alerting, or AI flows

**Checkpoint**: All user stories should now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Run `gofmt` and `goimports` on touched Go files
- [ ] TXXX Execute targeted `go test` packages and required scripts or
          benchmarks
- [ ] TXXX [P] Validate Docker Compose, Helm, Kubernetes, or client workflows
          if affected
- [ ] TXXX Update `README.md`, `doc/`, and `.specify` artifacts to match the
          delivered behavior
- [ ] TXXX Capture any skipped validation or follow-up debt explicitly in review
          notes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but should be independently testable

### Within Each User Story

- Add the strongest relevant verification before or alongside implementation
- Contracts and config before regenerated outputs and client wiring
- Core implementation before cross-service integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All verification tasks for a user story marked [P] can run in parallel
- Different packages within a story marked [P] can run in parallel when they do
  not touch the same files
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch verification for User Story 1 together:
Task: "Add tests in internal/[area]/[name]_test.go"
Task: "Add smoke coverage in test/[fixture] or scripts/[tool]/..."

# Launch independent implementation work together:
Task: "Implement core logic in internal/[bounded-context]/..."
Task: "Update entrypoint or contract in cmd/[service]/main.go or api/proto/v1/[contract].proto"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Regenerate `api/gen/v1/` whenever `.proto` files change
- Sync `configs/` and `deployments/` whenever runtime inputs or service topology change
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
