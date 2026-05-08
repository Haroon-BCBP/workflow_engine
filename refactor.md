# Refactoring Log

## Project Structure & Dependency Injection

- **Modularity**: Split the monolithic `handler.go` into functional groups: `workflow_handler.go`, `document_handler.go`, `iam_handler.go`, and `health_handler.go`.
- **Abstraction**: Decoupled `WorkflowService` using an interface and moved the implementation to `workflow_service.go`.
- **DI Container**: Introduced an `App` struct in `cmd/api/app.go` to manage application dependencies (DB, Temporal, IAM, Services) and handle unified cleanup.
- **Configuration**: Centralized environment variable management in a dedicated `config` package.
- **Persistence**: initial setup for database connection logic in `db.go` to support future database migrations.

## Workload Calculation & State Organization

- **Optimization**: Refactored `calculateRunWorkload` in `utils.go` to use a pre-calculated `Workload` field in the `WorkflowState`. This reduced the complexity of calculating active user tasks across all runs from $O(N^3)$ to $O(1)$ at query time.
- **Organization**: Moved helper methods (`IsActive`, `AllAssignees`, `UpdateWorkload`) from `dsl/types.go` to a new `dsl/state.go` file to maintain a clean domain model definition.
- **Optimization**: Refactored `isUserAssignedToRun` in `utils.go` to use a map-based lookup for department IDs, improving performance for run visibility checks.
- **Cleanup**: Removed the fallback loop logic in `utils.go` after ensuring all workflows maintain the `Workload` state.

## Workflow Engine Restructuring & Advanced Refactoring

- **Package Migration**: Replaced the generic `internal/dsl` package with a dedicated `internal/workflow` package for better domain clarity.
- **Modularity**: Split the monolithic `workflow.go` into specialized files:
  - `engine.go`: Main orchestrator and execution plan logic.
  - `steps.go`: Decoupled handlers for Sequential, Parallel, and Exclusive steps.
  - `stages.go`: Isolated department and stage processing logic.
  - `signals.go`: Centralized signal handling with a new generic `relaySignal` utility.
  - `state.go`: Encapsulated state transitions and assignment logic (`ApplyAssignments`).
- **Logic Refinement**:
  - **Signal Relaying**: Reduced boilerplate using a type-safe generic signal distributor.
  - **Parallel Execution**: Simplified the `runParallel` inner loop for better readability and performance.
  - **Declarative Main Loop**: Refactored the core workflow loop to be more declarative and maintainable.
- **Stability**: Fixed undefined type errors and restored missing administrative routing functions identified during build verification.
