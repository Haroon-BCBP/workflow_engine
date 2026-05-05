import React, { useEffect, useRef, useState } from "react";
// @ts-ignore
import Modeler from "bpmn-js/lib/Modeler";
import {
  BpmnPropertiesPanelModule,
  BpmnPropertiesProviderModule,
  ZeebePropertiesProviderModule,
} from "bpmn-js-properties-panel";
// @ts-ignore
import zeebeModdleDescriptor from "zeebe-bpmn-moddle/resources/zeebe.json";
// @ts-ignore
import {
  ElementTemplatesCoreModule,
  ElementTemplatesPropertiesProviderModule,
} from "bpmn-js-element-templates";
import { api } from "../api/client";

import "bpmn-js/dist/assets/diagram-js.css";
import "bpmn-js/dist/assets/bpmn-js.css";
import "bpmn-js/dist/assets/bpmn-font/css/bpmn.css";
import "@bpmn-io/properties-panel/dist/assets/properties-panel.css";

import "./BpmnModeler.css";
// @ts-ignore
import initialDiagram from "./initial-diagram.bpmn?raw";
// @ts-ignore
import templates from "./templates.json";

interface Props {
  onWorkflowStarted?: (workflowId: string) => void;
}

const BpmnModelerComponent: React.FC<Props> = ({ onWorkflowStarted }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const propertiesRef = useRef<HTMLDivElement>(null);
  const modelerRef = useRef<any>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [submitting, setSubmitting] = useState(false);
  const [statusMsg, setStatusMsg] = useState<string | null>(null);

  useEffect(() => {
    if (containerRef.current && propertiesRef.current && !modelerRef.current) {
      modelerRef.current = new Modeler({
        container: containerRef.current,
        propertiesPanel: { parent: propertiesRef.current },
        additionalModules: [
          BpmnPropertiesPanelModule,
          BpmnPropertiesProviderModule,
          ZeebePropertiesProviderModule,
          ElementTemplatesCoreModule,
          ElementTemplatesPropertiesProviderModule,
        ],
        moddleExtensions: { zeebe: zeebeModdleDescriptor },
        keyboard: { bindTo: globalThis },
        elementTemplates: templates,
      });
      modelerRef.current.importXML(initialDiagram).catch((err: any) => {
        console.error("Error rendering BPMN Diagram", err);
      });
    }
    return () => {
      if (modelerRef.current) {
        modelerRef.current.destroy();
        modelerRef.current = null;
      }
    };
  }, []);

  const handleDownload = async () => {
    if (!modelerRef.current) return;
    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      const blob = new Blob([xml], {
        type: "application/bpmn20-xml;charset=UTF-8",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "diagram.bpmn";
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Error saving XML", err);
    }
  };

  const handleSubmit = async () => {
    if (!modelerRef.current) return;
    setSubmitting(true);
    setStatusMsg(null);
    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      const result = await api.submit(xml);
      setStatusMsg(`✓ Workflow started: ${result.workflow_id}`);
      if (onWorkflowStarted) onWorkflowStarted(result.workflow_id);
    } catch (err: any) {
      setStatusMsg("Error: " + err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleFileLoad = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    try {
      const xml = await file.text();
      if (modelerRef.current && xml) {
        await modelerRef.current.importXML(xml);
      }
    } catch (err) {
      console.error("Error importing XML", err);
      alert("Error importing XML. Please ensure it's a valid BPMN file.");
    }
    event.target.value = "";
  };

  return (
    <div className="modeler-container">
      <div className="button-bar">
        <button
          className="btn-primary"
          onClick={handleDownload}
          id="btn-download-xml"
        >
          Download XML
        </button>
        <button
          className="btn-primary"
          onClick={() => fileInputRef.current?.click()}
          id="btn-load-xml"
        >
          Load XML
        </button>
        <input
          type="file"
          ref={fileInputRef}
          style={{ display: "none" }}
          accept=".bpmn,.xml"
          onChange={handleFileLoad}
        />
        <button
          className="btn-primary"
          onClick={handleSubmit}
          disabled={submitting}
          id="btn-submit-workflow"
        >
          {submitting ? "Starting…" : "▶ Start Workflow"}
        </button>
        {statusMsg && <span className="submit-status">{statusMsg}</span>}
      </div>

      <div className="modeler-hint">
        💡 <strong>Using templates:</strong> Add a <em>User Task</em> to the canvas → select it → click
        {" "}<strong>"Select template"</strong> in the right panel to apply the <em>Department Stage Task</em> template.
      </div>

      <div className="modeler-body">
        <div ref={containerRef} className="canvas-container" />
        <div ref={propertiesRef} className="properties-container" />
      </div>
    </div>
  );
};

export default BpmnModelerComponent;
