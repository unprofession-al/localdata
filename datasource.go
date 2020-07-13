package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	srcRegMu sync.Mutex
	srcReg   = SourceRegistry{}
)

type SourceRegistry map[string]func(string) (Source, error)

func (s SourceRegistry) List() []string {
	out := []string{}
	for name := range s {
		out = append(out, name)
	}
	return out
}

func RegisterSourceType(kind string, bootstrapFunc func(string) (Source, error)) {
	srcRegMu.Lock()
	defer srcRegMu.Unlock()
	if _, dup := srcReg[kind]; dup {
		panic("Register called twice for source type " + kind)
	}
	srcReg[kind] = bootstrapFunc
}

func NewSource(kind, config string) (Source, error) {
	bootstrapFunc, ok := srcReg[kind]
	if !ok {
		kinds := []string{}
		for k := range srcReg {
			kinds = append(kinds, k)
		}
		return nil, fmt.Errorf("Source type '%s' does not exist, must be one of the following: %s", kind, strings.Join(kinds, ", "))
	}
	return bootstrapFunc(config)
}

type Source interface {
	Sync(chan SyncStatus) (bool, error)
	Synced() (bool, error)
	CreateVolume(az string, tags []*ec2.TagSpecification) (string, error)
}

type SyncStatus struct {
	Progess int
	Done    int
}
