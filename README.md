# Hybrid BPMN Workflow Engine (POC)

A stateful, human-in-the-loop workflow engine built with **Go**, **Temporal**, and **React**. This POC demonstrates how to orchestrate construction processes using BPMN diagrams that are translated into Temporal DSL for reliable execution.

## Overview

The system allows administrators to model workflows in a visual BPMN editor, assign departments and roles to tasks, and execute them as stateful workflows. It supports:

- **Sequential & Parallel Execution**: Automatically detects parallel gateways in BPMN and groups departments for concurrent processing.
- **Human-in-the-Loop**: Stages (Prep → Review → Approve) require manual transitions and comments.
- **Admin Routing**: If a department rejects a package, an administrator can route the workflow back to any previous stage or terminate it.
- **Stateful Queries**: The UI polls Temporal for real-time status of all departments and stages.

---

## How to Run

### 1. Start Temporal Server

In a new terminal window, start the local Temporal dev server:

```bash
temporal server start-dev
```

*The server will be available at `localhost:7233` and the Web UI at `localhost:8233`.*

### 2. Run the Backend

The backend consists of an **API Server** and a **Workflow Worker**.

**Using Air (Live Reload):**

```bash
# Terminal 1: API
cd backend
make watch-api

```

```sh
# Terminal 2: Worker
make watch-worker
```

### 3. Run the Frontend

```bash
cd UI
npm install
npm run dev
```

*The UI will be available at `http://localhost:5173`.*

---

## Using the POC

1. **Modeler**: Go to the **BPMN Modeler** tab. You'll see a pre-loaded diagram with 8 departments.
2. **Assign Roles**: Select a task (e.g., "Prep" in Design) and use the **Department Stage Task** template in the properties panel to set the `dept_id` and `role`.
3. **Submit**: Click **Submit Workflow**. This parses the BPMN, generates a YAML DSL, and starts a Temporal workflow.
4. **Dashboard**: Go to the **Dashboard** tab. Select your workflow from the sidebar.
5. **Progress**:
    - Add comments to stages. (Note: Adding a comment during the **Review** stage will return the department to the **Prep** stage when attempting to advance).
    - Click **Send for Review** / **Send for Approval** to advance.
    - Click **Reject** to trigger an Admin Routing event.
    - (Admin) Use the **Route Rejection** button to send the workflow back to an earlier stage.

---


## Issues/ Pending

1. Failures at parallel gateways: If a stage is rejected at a parallel gateway We need to consider how to send it previous stage, and also how to handle the other parallel stage which may or may not have been done. For now the workflow is marked failed if it fails at parallel stage, and current idea is to send back/ restart the whole workflow from last non parallel stage
2. If we need to send from some stage to previous one but the previous one is a parallel stage, should that be considered both stages redo or single just the one.
