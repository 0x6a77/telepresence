package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/telepresence2/pkg/rpc"
)

func TestMechanismHelpers(t *testing.T) {
	a := assert.New(t)

	testMechs := GetTestMechanisms(t)
	testAgents := GetTestAgents(t)

	empty := []*rpc.AgentInfo_Mechanism{}
	oss := testAgents["hello"].Mechanisms
	plus := testAgents["helloPro"].Mechanisms
	sameAsPlus := []*rpc.AgentInfo_Mechanism{testMechs["http"], testMechs["grpc"], testMechs["tcp"]}
	plus2 := []*rpc.AgentInfo_Mechanism{testMechs["tcp"], testMechs["grpc"], testMechs["httpv2"]}
	bogus := []*rpc.AgentInfo_Mechanism{testMechs["tcp"], testMechs["http"], testMechs["httpv2"]} // 2 http

	a.False(mechanismsAreTheSame(empty, empty))
	a.False(mechanismsAreTheSame(oss, plus))
	a.False(mechanismsAreTheSame(plus, plus2))
	a.False(mechanismsAreTheSame(plus, bogus))
	a.True(mechanismsAreTheSame(plus, sameAsPlus))
	a.True(mechanismsAreTheSame(testAgents["demo1"].Mechanisms, testAgents["demo2"].Mechanisms))
	a.True(mechanismsAreTheSame(oss, []*rpc.AgentInfo_Mechanism{testMechs["tcp"]}))
}

func TestAgentHelpers(t *testing.T) {
	a := assert.New(t)

	testAgents := GetTestAgents(t)
	helloAgent := testAgents["hello"]
	helloProAgent := testAgents["helloPro"]
	demoAgent1 := testAgents["demo1"]
	demoAgent2 := testAgents["demo2"]

	a.True(agentsAreCompatible([]*rpc.AgentInfo{demoAgent1, demoAgent2}))
	a.True(agentsAreCompatible([]*rpc.AgentInfo{helloAgent}))
	a.True(agentsAreCompatible([]*rpc.AgentInfo{helloProAgent}))
	a.False(agentsAreCompatible([]*rpc.AgentInfo{}))
	a.False(agentsAreCompatible([]*rpc.AgentInfo{helloAgent, helloProAgent}))
}
