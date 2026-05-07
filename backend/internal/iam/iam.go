package iam

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)


func Load(path string) (*IAM, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("iam: read %s: %w", path, err)
	}
	var cfg iamConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("iam: parse yaml: %w", err)
	}
	return &IAM{cfg: cfg}, nil
}

// role is one of: "preparer", "reviewer", "approver".
func (i *IAM) GetAssignees(deptID, role string) ([]User, error) {
	dept, ok := i.cfg.Departments[deptID]
	if !ok {
		return nil, fmt.Errorf("iam: unknown department %q", deptID)
	}
	users, ok := dept[role]
	if !ok {
		return nil, fmt.Errorf("iam: unknown role %q in department %q", role, deptID)
	}
	return users, nil
}

func (i *IAM) GetAdmins() []User {
	return i.cfg.Admins
}

func (i *IAM) IsAdmin(userID string) bool {
	for _, a := range i.cfg.Admins {
		if a.UserID == userID {
			return true
		}
	}
	return false
}

func (i *IAM) AllDeptRoles() map[string]map[string][]User {
	return i.cfg.Departments
}
func (i *IAM) GetUserDepartments(userID string) []string {
	var depts []string
	for deptID, roles := range i.cfg.Departments {
		found := false
		for _, users := range roles {
			for _, u := range users {
				if u.UserID == userID {
					depts = append(depts, deptID)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	return depts
}
