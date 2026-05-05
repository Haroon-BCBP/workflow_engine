package bpmn

import (
	"encoding/xml"
)

type Definitions struct {
	XMLName   xml.Name  `xml:"definitions"`
	Processes []Process `xml:"process"`
}

type Process struct {
	ID          string         `xml:"id,attr"`
	Name        string         `xml:"name,attr"`
	LaneSet     *LaneSet       `xml:"laneSet"`
	UserTasks   []UserTask     `xml:"userTask"`
	StartEvents []StartEvent   `xml:"startEvent"`
	EndEvents   []EndEvent     `xml:"endEvent"`
	Gateways    []Gateway      `xml:"parallelGateway"`
	ExcGateways []Gateway      `xml:"exclusiveGateway"`
	Flows       []SequenceFlow `xml:"sequenceFlow"`
}

type LaneSet struct {
	Lanes []Lane `xml:"lane"`
}

type Lane struct {
	ID       string   `xml:"id,attr"`
	Name     string   `xml:"name,attr"`
	FlowRefs []string `xml:"flowNodeRef"`
}

type UserTask struct {
	ID                string       `xml:"id,attr"`
	Name              string       `xml:"name,attr"`
	ExtensionElements *ExtElements `xml:"extensionElements"`
}

type StartEvent struct {
	ID string `xml:"id,attr"`
}

type EndEvent struct {
	ID string `xml:"id,attr"`
}

type Gateway struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type SequenceFlow struct {
	ID     string `xml:"id,attr"`
	Source string `xml:"sourceRef,attr"`
	Target string `xml:"targetRef,attr"`
}

type ExtElements struct {
	Properties *ZeebeProperties `xml:"properties"`
}

type ZeebeProperties struct {
	Properties []ZeebeProperty `xml:"property"`
}

type ZeebeProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type Parser struct{}
