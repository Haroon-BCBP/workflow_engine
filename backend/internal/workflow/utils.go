package workflow

func findDept(def WorkflowDef, id string) *DepartmentDef {
	for i, d := range def.Departments {
		if d.ID == id {
			return &def.Departments[i]
		}
	}
	return nil
}

func findStageIndex(dept DepartmentDef, stageType StageType) int {
	for i, s := range dept.Stages {
		if s.Type == stageType {
			return i
		}
	}
	return 0
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
