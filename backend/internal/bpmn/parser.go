package bpmn

import (
	"encoding/xml"
	"fmt"
	"strings"

	engine "github.com/Haroon-BCBP/workflow_engine/internal/workflow"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)




func (p *Parser) ParseXML(xmlData []byte) (*engine.WorkflowDef, error) {
	var defs Definitions
	if err := xml.Unmarshal(xmlData, &defs); err != nil {
		return nil, fmt.Errorf("bpmn: xml unmarshal: %w", err)
	}
	if len(defs.Processes) == 0 {
		return nil, fmt.Errorf("bpmn: no process found in XML")
	}

	proc := defs.Processes[0]

	taskLane := buildTaskLaneMap(proc)

	deptOrder := laneOrder(proc)

	deptMap := make(map[string]*engine.DepartmentDef)
	for _, task := range proc.UserTasks {
		props := zeebeProps(task)
		deptID := props["dept_id"]
		stageType := props["stage_type"]
		role := props["role"]

		if deptID == "" {
			if lane, ok := taskLane[task.ID]; ok {
				deptID = normalizeDeptID(lane)
			}
		}
		if stageType == "" {
			stageType = inferStageType(task.Name)
		}
		if role == "" {
			role = inferRole(stageType)
		}

		if deptID == "" || stageType == "" {
			continue // skip tasks we can't classify
		}

		if _, ok := deptMap[deptID]; !ok {
			deptMap[deptID] = &engine.DepartmentDef{
				ID:    deptID,
				Label: labelFromID(deptID),
			}
		}
		deptMap[deptID].Stages = append(deptMap[deptID].Stages, engine.StageDef{
			Type:            engine.StageType(stageType),
			Activity:        activityName(stageType),
			Role:            role,
			RequiresComment: stageType == "review",
		})
	}

	for _, dept := range deptMap {
		dept.Stages = sortStages(dept.Stages)
	}

	var departments []engine.DepartmentDef
	seen := map[string]bool{}
	for _, id := range deptOrder {
		if d, ok := deptMap[id]; ok && !seen[id] {
			departments = append(departments, *d)
			seen[id] = true
		}
	}
	for id, d := range deptMap {
		if !seen[id] {
			departments = append(departments, *d)
		}
	}

	execution := buildExecutionPlan(proc, deptOrder)

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

func buildTaskLaneMap(proc Process) map[string]string {
	m := make(map[string]string)
	if proc.LaneSet == nil {
		return m
	}
	for _, lane := range proc.LaneSet.Lanes {
		for _, ref := range lane.FlowRefs {
			m[ref] = lane.Name
		}
	}
	return m
}

func laneOrder(proc Process) []string {
	var order []string
	seen := map[string]bool{}

	taskToDeptID := make(map[string]string)
	for _, t := range proc.UserTasks {
		if props := zeebeProps(t); props["dept_id"] != "" {
			taskToDeptID[t.ID] = props["dept_id"]
		}
	}

	if proc.LaneSet != nil {
		for _, lane := range proc.LaneSet.Lanes {
			id := ""
			for _, ref := range lane.FlowRefs {
				if dID, ok := taskToDeptID[ref]; ok {
					id = dID
					break
				}
			}

			if id == "" {
				id = normalizeDeptID(lane.Name)
			}

			if id != "" && !seen[id] {
				order = append(order, id)
				seen[id] = true
			}
		}
	}
	return order
}

func zeebeProps(task UserTask) map[string]string {
	props := make(map[string]string)
	if task.ExtensionElements == nil || task.ExtensionElements.Properties == nil {
		return props
	}
	for _, p := range task.ExtensionElements.Properties.Properties {
		props[p.Name] = p.Value
	}
	return props
}

func normalizeDeptID(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return s
}

func labelFromID(id string) string {
	return cases.Title(language.English).String(strings.ReplaceAll(id, "_", " "))
}

func inferStageType(taskName string) string {
	lower := strings.ToLower(taskName)
	switch {
	case strings.HasPrefix(lower, "prep"):
		return "prep"
	case strings.HasPrefix(lower, "review"):
		return "review"
	case strings.HasPrefix(lower, "approv"):
		return "approve"
	}
	return ""
}

func inferRole(stageType string) string {
	switch stageType {
	case "prep":
		return "preparer"
	case "review":
		return "reviewer"
	case "approve":
		return "approver"
	}
	return ""
}

func activityName(_ string) string {
	return "StageStartedActivity"
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

// Algorithm:
//  1. Build directed adjacency lists from <sequenceFlow> elements.
//  2. Classify each <parallelGateway> as split (>1 outgoing) or join (>1 incoming).
//  3. For each split gateway, BFS along every outgoing branch until a join is reached.
//     All dept IDs encountered across all branches of the same split form one parallel group.
//  4. Walk deptOrder (lane order from the XML) to emit steps:
//     sequential depts go into {sequential:[...]}, parallel groups go into {parallel:[...]}.
func buildExecutionPlan(proc Process, deptOrder []string) engine.ExecutionPlan {
	if len(deptOrder) == 0 {
		return engine.ExecutionPlan{}
	}
	if len(proc.Gateways) == 0 {
		return engine.ExecutionPlan{
			Steps: []engine.ExecutionStep{{Sequential: deptOrder}},
		}
	}

	outgoing := make(map[string][]string) // nodeID → []targetIDs
	incoming := make(map[string][]string) // nodeID → []sourceIDs
	for _, flow := range proc.Flows {
		outgoing[flow.Source] = append(outgoing[flow.Source], flow.Target)
		incoming[flow.Target] = append(incoming[flow.Target], flow.Source)
	}

	splitGWs := make(map[string]bool) // diverging parallel: >1 outgoing edges
	excSplitGWs := make(map[string]bool) // diverging exclusive
	joinGWs := make(map[string]bool)  // converging: >1 incoming edges
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
		return engine.ExecutionPlan{
			Steps: []engine.ExecutionStep{{Sequential: deptOrder}},
		}
	}

	taskLane := buildTaskLaneMap(proc)
	taskToDept := make(map[string]string)
	for _, t := range proc.UserTasks {
		if props := zeebeProps(t); props["dept_id"] != "" {
			taskToDept[t.ID] = props["dept_id"]
		}
	}
	for taskID, laneName := range taskLane {
		if _, ok := taskToDept[taskID]; !ok {
			if id := normalizeDeptID(laneName); id != "" {
				taskToDept[taskID] = id
			}
		}
	}

	// For each split gateway we BFS along every outgoing branch, stopping when
	// we reach a join gateway. All dept IDs found across all branches form one
	// parallel group (they all run concurrently between the split and join).
	type gatewayGroup struct {
		isExclusive bool
		depts       []string
	}
	var groups []gatewayGroup

	// BFS helper function
	findGroupDepts := func(splitID string) []string {
		groupDepts := make(map[string]bool)
		for _, branchStart := range outgoing[splitID] {
			visited := make(map[string]bool)
			queue := []string{branchStart}
			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				if visited[cur] {
					continue
				}
				visited[cur] = true

				if joinGWs[cur] {
					continue
				}

				if dept, ok := taskToDept[cur]; ok && dept != "" {
					groupDepts[dept] = true
				}

				for _, next := range outgoing[cur] {
					if !visited[next] {
						queue = append(queue, next)
					}
				}
			}
		}
		var depts []string
		for d := range groupDepts {
			depts = append(depts, d)
		}
		return depts
	}

	for splitID := range splitGWs {
		depts := findGroupDepts(splitID)
		if len(depts) >= 2 {
			groups = append(groups, gatewayGroup{isExclusive: false, depts: depts})
		}
	}
	for splitID := range excSplitGWs {
		depts := findGroupDepts(splitID)
		if len(depts) >= 2 {
			groups = append(groups, gatewayGroup{isExclusive: true, depts: depts})
		}
	}
	if len(groups) == 0 {
		return engine.ExecutionPlan{
			Steps: []engine.ExecutionStep{{Sequential: deptOrder}},
		}
	}

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

		pgDepts := make(map[string]bool)
		for _, d := range groups[gIdx].depts {
			pgDepts[d] = true
		}
		var orderedPar []string
		for _, d := range deptOrder {
			if pgDepts[d] {
				orderedPar = append(orderedPar, d)
			}
		}
		
		if groups[gIdx].isExclusive {
			steps = append(steps, engine.ExecutionStep{Exclusive: orderedPar})
		} else {
			steps = append(steps, engine.ExecutionStep{Parallel: orderedPar})
		}
		emittedGroups[gIdx] = true
	}

	if len(seqBuf) > 0 {
		steps = append(steps, engine.ExecutionStep{Sequential: seqBuf})
	}

	if len(steps) == 0 {
		steps = []engine.ExecutionStep{{Sequential: deptOrder}}
	}

	return engine.ExecutionPlan{Steps: steps}
}
