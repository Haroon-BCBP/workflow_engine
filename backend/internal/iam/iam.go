package iam

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type User struct {
	UserID string `yaml:"user_id"`
	Name   string `yaml:"name"`
}

type iamConfig struct {
	Departments map[string]map[string]User `yaml:"departments"`
	Admins      []User                     `yaml:"admins"`
}

type IAM struct {
	cfg iamConfig
}

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
func (i *IAM) GetAssignee(deptID, role string) (*User, error) {
	dept, ok := i.cfg.Departments[deptID]
	if !ok {
		return nil, fmt.Errorf("iam: unknown department %q", deptID)
	}
	user, ok := dept[role]
	if !ok {
		return nil, fmt.Errorf("iam: unknown role %q in department %q", role, deptID)
	}
	return &user, nil
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

func (i *IAM) AllDeptRoles() map[string]map[string]User {
	return i.cfg.Departments
}
