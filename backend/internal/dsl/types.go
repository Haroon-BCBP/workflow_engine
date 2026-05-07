package dsl

type WorkflowDef struct {
	Name        string          `yaml:"name"`
	Version     string          `yaml:"version"`
	TaskQueue   string          `yaml:"task_queue"`
	Departments []DepartmentDef `yaml:"departments"`
	Execution   ExecutionPlan   `yaml:"execution"`
}

type DepartmentDef struct {
	ID     string     `yaml:"id"`
	Label  string     `yaml:"label"`
	Stages []StageDef `yaml:"stages"`
}

type StageDef struct {
	Type            StageType `yaml:"type"`
	Activity        string    `yaml:"activity"`
	Role            string    `yaml:"role"`
	RequiresComment bool      `yaml:"requires_comment,omitempty"`
}

type StageType string

const (
	StagePrep    StageType = "prep"
	StageReview  StageType = "review"
	StageApprove StageType = "approve"
)

type ExecutionPlan struct {
	Steps []ExecutionStep `yaml:"steps" json:"steps"`
}

type ExecutionStep struct {
	Sequential []string `yaml:"sequential,omitempty" json:"sequential,omitempty"`
	Parallel   []string `yaml:"parallel,omitempty" json:"parallel,omitempty"`
	Exclusive  []string `yaml:"exclusive,omitempty" json:"exclusive,omitempty"`
}

type WorkflowState struct {
	WorkflowID  string                         `json:"workflow_id"`
	Name        string                         `json:"name"`
	CurrentStep int                            `json:"current_step"`
	Progress    map[string]*DepartmentProgress `json:"progress"` // keyed by dept ID
	Execution   ExecutionPlan                  `json:"execution"`
	Status      WorkflowStatus                 `json:"status"`
	RejectedBy  string                         `json:"rejected_by,omitempty"`
}

type WorkflowStatus string

const (
	WorkflowPendingAssignment WorkflowStatus = "pending_assignment"
	WorkflowRunning           WorkflowStatus = "running"
	WorkflowApproved          WorkflowStatus = "approved"
	WorkflowRejected          WorkflowStatus = "rejected"
	WorkflowPaused            WorkflowStatus = "paused"     // waiting for admin routing after rejection
	WorkflowPausedXOR         WorkflowStatus = "paused_xor" // waiting for admin to select XOR path
)

type DepartmentProgress struct {
	DeptID             string                `json:"dept_id"`
	Label              string                `json:"label"`
	CurrentStage       StageType             `json:"current_stage"`
	StageStatus        StageStatus           `json:"stage_status"`
	StageAssignees     map[StageType]string  `json:"stage_assignees"`
	StageAssigneeNames map[StageType]string  `json:"stage_assignee_names"`
	HasComment         bool                  `json:"has_comment"`
	Comments           []Comment             `json:"comments"`
}

type StageStatus string

const (
	StageStatusPending    StageStatus = "pending" // not yet reached
	StageStatusInProgress StageStatus = "in_progress"
	StageStatusDone       StageStatus = "done"
	StageStatusRejected   StageStatus = "rejected"
)

type Comment struct {
	UserID string    `json:"user_id"`
	Text   string    `json:"text"`
	Stage  StageType `json:"stage"`
}

type TransitionSignal struct {
	DeptID  string    `json:"dept_id"`
	ToStage StageType `json:"to_stage"` // "review", "approve", or "reject"
	UserID  string    `json:"user_id"`
}

type CommentSignal struct {
	DeptID string    `json:"dept_id"`
	Stage  StageType `json:"stage"`
	UserID string    `json:"user_id"`
	Text   string    `json:"text"`
}

type DocumentSignal struct {
	DeptID string    `json:"dept_id"`
	Stage  StageType `json:"stage"`
}

type AdminRoutingSignal struct {
	Action  string    `json:"action"` // "goto" | "terminate"
	DeptID  string    `json:"dept_id,omitempty"`
	Stage   StageType `json:"stage,omitempty"`
	AdminID string    `json:"admin_id"`
}

type Assignment struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

type AdminStartSignal struct {
	Assignments map[string]map[StageType]Assignment `json:"assignments"` // deptID -> stage -> Assignment
	AdminID     string                              `json:"admin_id"`
}

const (
	TransitionChannel   = "StageTransitionChannel"
	CommentChannel      = "CommentChannel"
	DocumentChannel     = "DocumentChannel"
	AdminRoutingChannel = "AdminRoutingChannel"
	AdminStartChannel   = "AdminStartChannel"
	QueryStatus         = "GetWorkflowStatus"
	TaskQueue           = "workflow-engine-queue-v2"
)
