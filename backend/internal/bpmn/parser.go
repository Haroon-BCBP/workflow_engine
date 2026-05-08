package bpmn

import (
	"encoding/xml"
	"fmt"
	"strings"

	engine "github.com/Haroon-BCBP/workflow_engine/internal/workflow"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var departmentLabels = map[string]string{
	"design":        "Design / Engineering",
	"contracts":     "Contract Management",
	"procurement":   "Procurement",
	"planning":      "Planning",
	"logistics":     "Logistics",
	"hse":           "HSE",
	"construction":  "Construction",
	"qa":            "QA / QC",
	"commissioning": "Commissioning",
}

func (p *Parser) ParseXML(xmlData []byte) (*engine.WorkflowDef, error) {
	var defs Definitions
	if err := xml.Unmarshal(xmlData, &defs); err != nil {
		return nil, fmt.Errorf("bpmn: xml unmarshal: %w", err)
	}
	if len(defs.Processes) == 0 {
		return nil, fmt.Errorf("bpmn: no process found in XML")
	}

	proc := defs.Processes[0]
	deptMap := p.buildDepartmentMap(proc)
	deptOrder := p.getLaneOrder(proc)

	var departments []engine.DepartmentDef
	seen := map[string]bool{}
	for _, id := range deptOrder {
		if d, ok := deptMap[id]; ok && !seen[id] {
			departments = append(departments, *d)
			seen[id] = true
		}
	}

	execution := p.buildExecutionPlan(proc, deptOrder)

	workflowName := proc.Name
	if workflowName == "" {
		workflowName = "Untitled Workflow"
	}

	return &engine.WorkflowDef{
		Name:        workflowName,
		Version:     "1.0",
		TaskQueue:   engine.TaskQueue,
		Departments: departments,
		Execution:   execution,
	}, nil
}

func (p *Parser) buildDepartmentMap(proc Process) map[string]*engine.DepartmentDef {
	deptMap := make(map[string]*engine.DepartmentDef)
	for _, task := range proc.UserTasks {
		props := getZeebeProps(task)
		deptID, stageType, role := props["dept_id"], props["stage_type"], props["role"]

		if deptID == "" || stageType == "" {
			continue
		}

		dept, ok := deptMap[deptID]
		if !ok {
			dept = &engine.DepartmentDef{
				ID:    deptID,
				Label: p.getDeptLabel(deptID),
			}
			deptMap[deptID] = dept
		}

		dept.Stages = append(dept.Stages, engine.StageDef{
			Type:            engine.StageType(stageType),
			Activity:        "StageStartedActivity",
			Role:            role,
			RequiresComment: p.isCommentRequired(stageType, props),
		})
	}

	for _, dept := range deptMap {
		dept.Stages = sortStages(dept.Stages)
	}
	return deptMap
}

func (p *Parser) getDeptLabel(id string) string {
	if label, ok := departmentLabels[id]; ok {
		return label
	}
	return cases.Title(language.English).String(strings.ReplaceAll(id, "_", " "))
}

func (p *Parser) isCommentRequired(stageType string, props map[string]string) bool {
	if val, ok := props["requires_comment"]; ok {
		return val == "true"
	}
	return stageType == "review"
}

func (p *Parser) getLaneOrder(proc Process) []string {
	if proc.LaneSet == nil {
		return nil
	}

	taskToDept := make(map[string]string)
	for _, t := range proc.UserTasks {
		if dID := getZeebeProps(t)["dept_id"]; dID != "" {
			taskToDept[t.ID] = dID
		}
	}

	var order []string
	seen := map[string]bool{}
	for _, lane := range proc.LaneSet.Lanes {
		for _, ref := range lane.FlowRefs {
			if dID, ok := taskToDept[ref]; ok && !seen[dID] {
				order = append(order, dID)
				seen[dID] = true
				break
			}
		}
	}
	return order
}

func getZeebeProps(task UserTask) map[string]string {
	props := make(map[string]string)
	if task.ExtensionElements == nil || task.ExtensionElements.Properties == nil {
		return props
	}
	for _, p := range task.ExtensionElements.Properties.Properties {
		props[p.Name] = p.Value
	}
	return props
}

var stageOrder = map[engine.StageType]int{
	engine.StagePrep:    0,
	engine.StageReview:  1,
	engine.StageApprove: 2,
}

func sortStages(stages []engine.StageDef) []engine.StageDef {
	sorted := make([]engine.StageDef, len(stages))
	copy(sorted, stages)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if stageOrder[sorted[i].Type] > stageOrder[sorted[j].Type] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

type adjMap map[string][]string

func (p *Parser) buildExecutionPlan(proc Process, deptOrder []string) engine.ExecutionPlan {
	if len(deptOrder) == 0 {
		return engine.ExecutionPlan{}
	}
	if len(proc.Gateways) == 0 && len(proc.ExcGateways) == 0 {
		return engine.ExecutionPlan{Steps: []engine.ExecutionStep{{Sequential: deptOrder}}}
	}

	outgoing, incoming := make(adjMap), make(adjMap)
	for _, flow := range proc.Flows {
		outgoing[flow.Source] = append(outgoing[flow.Source], flow.Target)
		incoming[flow.Target] = append(incoming[flow.Target], flow.Source)
	}

	splitGWs, excSplitGWs, joinGWs := make(map[string]bool), make(map[string]bool), make(map[string]bool)
	for _, gw := range proc.Gateways {
		if len(outgoing[gw.ID]) > 1 {
			splitGWs[gw.ID] = true
		}
		if len(incoming[gw.ID]) > 1 {
			joinGWs[gw.ID] = true
		}
	}
	for _, gw := range proc.ExcGateways {
		if len(outgoing[gw.ID]) > 1 {
			excSplitGWs[gw.ID] = true
		}
		if len(incoming[gw.ID]) > 1 {
			joinGWs[gw.ID] = true
		}
	}

	if len(splitGWs) == 0 && len(excSplitGWs) == 0 {
		return engine.ExecutionPlan{Steps: []engine.ExecutionStep{{Sequential: deptOrder}}}
	}

	taskToDept := make(map[string]string)
	for _, t := range proc.UserTasks {
		if dID := getZeebeProps(t)["dept_id"]; dID != "" {
			taskToDept[t.ID] = dID
		}
	}

	var groups []gatewayGroup
	for splitID := range splitGWs {
		if depts := p.findGroupDepts(splitID, outgoing, joinGWs, taskToDept); len(depts) >= 2 {
			groups = append(groups, gatewayGroup{isExclusive: false, depts: depts})
		}
	}
	for splitID := range excSplitGWs {
		if depts := p.findGroupDepts(splitID, outgoing, joinGWs, taskToDept); len(depts) >= 2 {
			groups = append(groups, gatewayGroup{isExclusive: true, depts: depts})
		}
	}

	if len(groups) == 0 {
		return engine.ExecutionPlan{Steps: []engine.ExecutionStep{{Sequential: deptOrder}}}
	}

	return p.assembleSteps(deptOrder, groups)
}

type gatewayGroup struct {
	isExclusive bool
	depts       []string
}

func (p *Parser) findGroupDepts(splitID string, outgoing adjMap, joinGWs map[string]bool, taskToDept map[string]string) []string {
	groupDepts := make(map[string]bool)
	for _, branchStart := range outgoing[splitID] {
		visited := make(map[string]bool)
		queue := []string{branchStart}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if visited[cur] || joinGWs[cur] {
				continue
			}
			visited[cur] = true

			if dept, ok := taskToDept[cur]; ok && dept != "" {
				groupDepts[dept] = true
			}
			queue = append(queue, outgoing[cur]...)
		}
	}
	var depts []string
	for d := range groupDepts {
		depts = append(depts, d)
	}
	return depts
}

func (p *Parser) assembleSteps(deptOrder []string, groups []gatewayGroup) engine.ExecutionPlan {
	parallelGroupOf := make(map[string]int)
	for _, d := range deptOrder {
		parallelGroupOf[d] = -1
	}
	for gIdx, pg := range groups {
		for _, d := range pg.depts {
			parallelGroupOf[d] = gIdx
		}
	}

	var steps []engine.ExecutionStep
	var seqBuf []string
	emittedGroups := make(map[int]bool)

	for _, dept := range deptOrder {
		gIdx := parallelGroupOf[dept]
		if gIdx == -1 {
			seqBuf = append(seqBuf, dept)
			continue
		}
		if emittedGroups[gIdx] {
			continue
		}

		if len(seqBuf) > 0 {
			steps = append(steps, engine.ExecutionStep{Sequential: seqBuf})
			seqBuf = nil
		}

		pg := groups[gIdx]
		pgDepts := make(map[string]bool)
		for _, d := range pg.depts {
			pgDepts[d] = true
		}
		var orderedPar []string
		for _, d := range deptOrder {
			if pgDepts[d] {
				orderedPar = append(orderedPar, d)
			}
		}

		if pg.isExclusive {
			steps = append(steps, engine.ExecutionStep{Exclusive: orderedPar})
		} else {
			steps = append(steps, engine.ExecutionStep{Parallel: orderedPar})
		}
		emittedGroups[gIdx] = true
	}

	if len(seqBuf) > 0 {
		steps = append(steps, engine.ExecutionStep{Sequential: seqBuf})
	}
	return engine.ExecutionPlan{Steps: steps}
}
