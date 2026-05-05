package bpmn_test

import (
	"testing"

	"github.com/Haroon-BCBP/workflow_engine/internal/bpmn"
)

// sequentialXML has two lanes (design, planning) with no parallel gateways.
const sequentialXML = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"
             xmlns:zeebe="http://camunda.org/schema/zeebe/1.0"
             id="definitions" targetNamespace="http://bpmn.io/schema/bpmn">
  <process id="proc1" name="Sequential Test">
    <laneSet>
      <lane id="lane_design" name="Design">
        <flowNodeRef>task_d_prep</flowNodeRef>
        <flowNodeRef>task_d_review</flowNodeRef>
        <flowNodeRef>task_d_approve</flowNodeRef>
      </lane>
      <lane id="lane_planning" name="Planning">
        <flowNodeRef>task_p_prep</flowNodeRef>
        <flowNodeRef>task_p_review</flowNodeRef>
        <flowNodeRef>task_p_approve</flowNodeRef>
      </lane>
    </laneSet>
    <userTask id="task_d_prep"    name="Prep Design">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="design"/><zeebe:property name="stage_type" value="prep"/><zeebe:property name="role" value="preparer"/></zeebe:properties></extensionElements>
    </userTask>
    <userTask id="task_d_review"  name="Review Design">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="design"/><zeebe:property name="stage_type" value="review"/><zeebe:property name="role" value="reviewer"/></zeebe:properties></extensionElements>
    </userTask>
    <userTask id="task_d_approve" name="Approve Design">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="design"/><zeebe:property name="stage_type" value="approve"/><zeebe:property name="role" value="approver"/></zeebe:properties></extensionElements>
    </userTask>
    <userTask id="task_p_prep"    name="Prep Planning">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="planning"/><zeebe:property name="stage_type" value="prep"/><zeebe:property name="role" value="preparer"/></zeebe:properties></extensionElements>
    </userTask>
    <userTask id="task_p_review"  name="Review Planning">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="planning"/><zeebe:property name="stage_type" value="review"/><zeebe:property name="role" value="reviewer"/></zeebe:properties></extensionElements>
    </userTask>
    <userTask id="task_p_approve" name="Approve Planning">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="planning"/><zeebe:property name="stage_type" value="approve"/><zeebe:property name="role" value="approver"/></zeebe:properties></extensionElements>
    </userTask>
    <sequenceFlow id="f1" sourceRef="task_d_approve" targetRef="task_p_prep"/>
  </process>
</definitions>`

// parallelXML has design→ split gateway → [qa, hse] → join gateway → commissioning
const parallelXML = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"
             xmlns:zeebe="http://camunda.org/schema/zeebe/1.0"
             id="definitions" targetNamespace="http://bpmn.io/schema/bpmn">
  <process id="proc2" name="Parallel Test">
    <laneSet>
      <lane id="lane_design"       name="Design">
        <flowNodeRef>task_d_approve</flowNodeRef>
      </lane>
      <lane id="lane_qa"           name="QA">
        <flowNodeRef>task_qa_prep</flowNodeRef>
      </lane>
      <lane id="lane_hse"          name="HSE">
        <flowNodeRef>task_hse_prep</flowNodeRef>
      </lane>
      <lane id="lane_commissioning" name="Commissioning">
        <flowNodeRef>task_com_prep</flowNodeRef>
      </lane>
    </laneSet>

    <userTask id="task_d_approve"  name="Approve Design">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="design"/><zeebe:property name="stage_type" value="approve"/><zeebe:property name="role" value="approver"/></zeebe:properties></extensionElements>
    </userTask>
    <userTask id="task_qa_prep"    name="Prep QA">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="qa"/><zeebe:property name="stage_type" value="prep"/><zeebe:property name="role" value="preparer"/></zeebe:properties></extensionElements>
    </userTask>
    <userTask id="task_hse_prep"   name="Prep HSE">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="hse"/><zeebe:property name="stage_type" value="prep"/><zeebe:property name="role" value="preparer"/></zeebe:properties></extensionElements>
    </userTask>
    <userTask id="task_com_prep"   name="Prep Commissioning">
      <extensionElements><zeebe:properties><zeebe:property name="dept_id" value="commissioning"/><zeebe:property name="stage_type" value="prep"/><zeebe:property name="role" value="preparer"/></zeebe:properties></extensionElements>
    </userTask>

    <!-- design → split → qa branch and hse branch → join → commissioning -->
    <parallelGateway id="gw_split" name="Split"/>
    <parallelGateway id="gw_join"  name="Join"/>

    <sequenceFlow id="f1" sourceRef="task_d_approve" targetRef="gw_split"/>
    <sequenceFlow id="f2" sourceRef="gw_split"       targetRef="task_qa_prep"/>
    <sequenceFlow id="f3" sourceRef="gw_split"       targetRef="task_hse_prep"/>
    <sequenceFlow id="f4" sourceRef="task_qa_prep"   targetRef="gw_join"/>
    <sequenceFlow id="f5" sourceRef="task_hse_prep"  targetRef="gw_join"/>
    <sequenceFlow id="f6" sourceRef="gw_join"        targetRef="task_com_prep"/>
  </process>
</definitions>`

func TestParseXML_Sequential(t *testing.T) {
	p := &bpmn.Parser{}
	def, err := p.ParseXML([]byte(sequentialXML))
	if err != nil {
		t.Fatalf("ParseXML error: %v", err)
	}
	if def.Name != "Sequential Test" {
		t.Errorf("expected name 'Sequential Test', got %q", def.Name)
	}
	if len(def.Departments) != 2 {
		t.Fatalf("expected 2 departments, got %d", len(def.Departments))
	}
	if len(def.Departments[0].Stages) != 3 {
		t.Errorf("design should have 3 stages, got %d", len(def.Departments[0].Stages))
	}
	if len(def.Execution.Steps) != 1 {
		t.Fatalf("expected 1 sequential step, got %d steps", len(def.Execution.Steps))
	}
	step := def.Execution.Steps[0]
	if len(step.Sequential) != 2 || step.Sequential[0] != "design" || step.Sequential[1] != "planning" {
		t.Errorf("sequential step = %v, want [design planning]", step.Sequential)
	}
}

func TestParseXML_Parallel(t *testing.T) {
	p := &bpmn.Parser{}
	def, err := p.ParseXML([]byte(parallelXML))
	if err != nil {
		t.Fatalf("ParseXML error: %v", err)
	}

	if len(def.Execution.Steps) != 3 {
		t.Fatalf("expected 3 execution steps, got %d: %+v", len(def.Execution.Steps), def.Execution.Steps)
	}

	step0 := def.Execution.Steps[0]
	if len(step0.Sequential) != 1 || step0.Sequential[0] != "design" {
		t.Errorf("step 0 sequential = %v, want [design]", step0.Sequential)
	}

	step1 := def.Execution.Steps[1]
	if len(step1.Parallel) != 2 {
		t.Errorf("step 1 parallel = %v, want [qa hse]", step1.Parallel)
	}
	parallelSet := map[string]bool{}
	for _, d := range step1.Parallel {
		parallelSet[d] = true
	}
	if !parallelSet["qa"] || !parallelSet["hse"] {
		t.Errorf("parallel step should contain qa and hse, got %v", step1.Parallel)
	}

	step2 := def.Execution.Steps[2]
	if len(step2.Sequential) != 1 || step2.Sequential[0] != "commissioning" {
		t.Errorf("step 2 sequential = %v, want [commissioning]", step2.Sequential)
	}
}

func TestParseXML_ZeebePropertiesTakesPrecedence(t *testing.T) {
	p := &bpmn.Parser{}
	def, err := p.ParseXML([]byte(sequentialXML))
	if err != nil {
		t.Fatalf("ParseXML error: %v", err)
	}
	for _, dept := range def.Departments {
		for _, stage := range dept.Stages {
			if stage.Role == "" {
				t.Errorf("dept %q stage %q has empty role", dept.ID, stage.Type)
			}
		}
	}
}
