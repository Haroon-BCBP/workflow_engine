# Workflow Engine POC: Technical Documentation

## 1. Project Overview

This project is a Proof of Concept (POC) for a flexible, BPMN-driven workflow engine. It leverages **Temporal.io** for durable workflow orchestration, **Go** for the backend logic, and **React** for the user interface.

The engine parses BPMN 2.0 diagrams, generates an execution plan, and executes it through a series of "Department Stages" (Preparation, Review, Approval).

---

## 2. Core Architecture

### 2.1 Backend Structure (`/backend`)

- **`cmd/api/`**: The REST API server using `go-chi`. It handles diagram submissions, workflow status queries, administrative routing, and document uploads.
- **`cmd/worker/`**: The Temporal worker that listens for workflow tasks and executes the orchestration logic.
- **`internal/bpmn/`**: Contains the BPMN parser. It converts standard BPMN XML into a custom Execution Plan (JSON/YAML) used by the Temporal workflow.
- **`internal/dsl/`**: Defines the workflow orchestration logic. This is the heart of the system where Temporal workflows and activities are defined.
- **`internal/iam/`**: A mock Identity and Access Management system that loads users, departments, and roles from `config/iam.yaml`.
- **`internal/repository/`**: Persistence layer using SQLite (`workflow_engine.db`) to store workflow runs, execution history, and documents.
- **`internal/service/`**: Business logic layer that coordinates between handlers, Temporal clients, and the repository.

### 2.2 Frontend Structure (`/UI`)

- **`src/bpmn-modeler/`**: Integrates `bpmn-js` to allow users to design and deploy workflows directly.
- **`src/dashboard/`**: The main interface for tracking active workflows, performing stage transitions, and managing documents.
- **`src/api/`**: Axios-based API client for communicating with the backend.

---

## 3. Workflow Working Logic

### 3.1 Diagram to Execution

1. A user creates or loads a diagram in the Modeler.
2. Upon clicking **"Start Workflow"**, the XML is sent to the backend.
3. The **BPMN Parser** identifies:
   - **Lanes**: Mapped to Departments.
   - **Tasks**: Mapped to Stages (Prep/Review/Approve) based on naming conventions.
   - **Gateways**: Mapped to Parallel or Exclusive execution blocks.
4. An **Execution Plan** is generated and saved.
5. A **Temporal Workflow** is started using this plan.

### 3.2 Execution Flow

- The workflow iterates through the Execution Plan steps.
- **Sequential Steps**: Departments are processed one by one.
- **Parallel Steps**: Multiple departments run concurrently. If any department rejects, the sibling branches are paused for administrative routing.
- **Exclusive Gateway (XOR)**: The workflow pauses and waits for an Admin to select a specific path to proceed.

### 3.3 Department Stages

Each department follows a standard lifecycle:

1. **Preparation**: User uploads documents or prepares the data.
2. **Review**: Reviewers check the work. If a comment is added, the stage is "rejected" back to Preparation.
3. **Approval**: Approvers give the final "OK".

---

## 4. BPMN Parser

The parser (`backend/internal/bpmn/parser.go`) is responsible for translating visual diagrams into executable code.

### 4.1 Department Mapping

The parser determines which department a task belongs to using a two-tier lookup:

1. **Custom Properties**: It first checks for `zeebe:property` entries with `name="dept_id"`. This is the preferred method as it is explicit.
2. **Lane Mapping**: If no property is found, it looks at the BPMN `Lane` containing the task and normalizes the lane name (e.g., "Design Dept" becomes `design`).

### 4.2 Grouping Logic (Parallel & XOR)

The parser uses a **Breadth-First Search (BFS)** algorithm to handle gateways:

- When it hits a **Split Gateway**, it starts a BFS on all outgoing branches.
- It continues until it reaches a **Join Gateway**.
- All tasks (departments) found between the split and join are grouped into a single `ExecutionStep`.
- **Parallel**: All departments in the group run concurrently.
- **Exclusive (XOR)**: The workflow pauses, and the group is presented as a list of choices for the administrator.

### 4.3 Sequential Buffering

Tasks that are not inside a gateway group are collected into "Sequential Buffers." These buffers ensure that the workflow preserves the top-to-bottom or left-to-right order defined in the diagram.

---

## 5. POC Simplifications

### 5.1 Identity & Access Management (IAM)

Hardcoded in `backend/config/iam.yaml`. Users are manually assigned to departments and roles for the demo. This would be replaced by an **OIDC or LDAP** integration. The system would query a real employee directory to verify if a user belongs to the "Design Reviewer" group.

### 5.2 Document Management

"Uploading" a document simply records a filename in the SQLite database. The "View Document" popup displays a **mock JSON blob** instead of a real file. Files would be stored in a cloud bucket (e.g., **AWS S3**) or a Dedicated Document Management System (DMS). The "Open" action would trigger a secure download link or an in-browser PDF viewer.

### 5.3 Administrative Logic

The "Admin" is just another user ID (`admin-1`) defined in the YAML. XOR path selection is a manual button click. XOR paths could be automated based on business rules (e.g., "If budget > $1M, route to Finance"). Administrative routing would be logged in a strict audit trail.

### 5.4 Persistence

Uses **SQLite** for metadata and a **local Temporal dev server** for orchestration.

---

## 6. Key Capabilities

### 6.1 Cross-Department Document Sharing

- Users can upload documents to specific stages.
- **Permissions**: only the owning department (or an Admin) can *open* and view the document content.

### 6.2 Administrative Routing

At any point of failure (rejection) or pause (XOR), an administrator can:

- Route the workflow to any department and any stage.
- Terminate the workflow entirely.
- This provides "human-in-the-loop" control over automated processes.

### 6.3 Role-Based Permissions (Mock)

- **Preparer**: Can advance from Prep to Review.
- **Reviewer**: Can advance from Review to Approve or send back with comments.
- **Approver**: Final sign-off.
- **Admin**: Full control over routing and global document access.

---

## 7. Data Model (SQLite)

- **`workflows`**: Stores BPMN XML and DSL YAML for each run.
- **`documents`**: Stores metadata for uploaded files, including department and stage of origin.

---

## 8. Extension Guide

To extend the engine:

1. **Adding new BPMN elements**: Update `internal/bpmn/parser.go` to handle new XML tags and map them to `dsl.ExecutionStep`.
2. **New Stage Logic**: Modify `internal/dsl/workflow.go` -> `processDepartment` to add custom logic (e.g., auto-approval rules).
3. **Actual IAM Integration**: Replace the `iam` package implementation with an OIDC/LDAP client.
