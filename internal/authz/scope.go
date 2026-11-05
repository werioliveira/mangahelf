package authz

import (
	"github.com/google/uuid"
)

type Scope struct {
	Kind    ScopeKind
	GroupID uuid.UUID
}

func GlobalScope() Scope {
	return Scope{Kind: Global}
}

func GroupScope(groupID uuid.UUID) Scope {
	return Scope{Kind: Group, GroupID: groupID}
}

func (s Scope) IsGlobal() bool {
	return s.Kind == Global
}

func (s Scope) IsGroup() bool {
	return s.Kind == Group
}

func (s Scope) String() string {
	if s.Kind == Group {
		return "group:" + s.GroupID.String()
	}
	return "global"
}