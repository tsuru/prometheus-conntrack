package testing

import "github.com/tsuru/prometheus-conntrack/workload"

type fakeEngine struct {
	name      string
	kind      string
	workloads []*workload.Workload
}

func (f fakeEngine) Name() string {
	return f.name
}

func (f fakeEngine) Kind() string {
	return f.kind
}

func (f fakeEngine) Workloads() ([]*workload.Workload, error) {
	return f.workloads, nil
}

// New creates a fake engine for tests propouses
func New(name, kind string, workloads []*workload.Workload) workload.Engine {
	return &fakeEngine{name, kind, workloads}
}
