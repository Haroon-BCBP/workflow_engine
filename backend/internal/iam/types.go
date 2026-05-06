package iam

type User struct {
	UserID string `yaml:"user_id" json:"user_id"`
	Name   string `yaml:"name" json:"name"`
}

type iamConfig struct {
	Departments map[string]map[string][]User `yaml:"departments"`
	Admins      []User                       `yaml:"admins"`
}

type IAM struct {
	cfg iamConfig
}
