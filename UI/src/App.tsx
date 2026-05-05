import React, { useState } from "react";
import BpmnModelerComponent from "./bpmn-modeler/BpmnModelerComponent";
import WorkflowDashboard from "./dashboard/WorkflowDashboard";
import "./App.css";

type View = "modeler" | "dashboard";

const App: React.FC = () => {
  const [view, setView] = useState<View>("modeler");
  const [activeWorkflowId, setActiveWorkflowId] = useState<string | null>(null);

  return (
    <div className="app-root">
      <header className="app-header">
        <div className="header-brand">
          <span className="brand-icon">⚙</span>
          <span className="brand-name">Workflow Engine</span>
        </div>
        <nav className="header-nav">
          <button
            className={`nav-btn ${view === "modeler" ? "active" : ""}`}
            onClick={() => setView("modeler")}
            id="nav-modeler"
          >
            BPMN Modeler
          </button>
          <button
            className={`nav-btn ${view === "dashboard" ? "active" : ""}`}
            onClick={() => setView("dashboard")}
            id="nav-dashboard"
          >
            Workflow Dashboard
          </button>
        </nav>
      </header>

      <main className="app-main">
        {view === "modeler" && (
          <BpmnModelerComponent
            onWorkflowStarted={(id) => {
              setActiveWorkflowId(id);
              setView("dashboard");
            }}
          />
        )}
        {view === "dashboard" && (
          <WorkflowDashboard initialWorkflowId={activeWorkflowId} />
        )}
      </main>
    </div>
  );
};

export default App;
