package task

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/sirupsen/logrus"
)

// Interface

type Param struct {
	Name  string
	Value interface{}
}

type Params []Param

type Artifacts map[string]interface{}

type OutputParser func([]byte) (Artifacts, error)

type Task interface {
	Do(params Params) (Artifacts, error)
}

// Implementation

type taskCmd struct {
	path   string
	l      *logrus.Entry
	parser OutputParser
}

func (t taskCmd) convertParams(params Params) []string {
	res := make([]string, 0, len(params))
	for _, p := range params {
		res = append(res, p.Name)
		if p.Value != nil {
			res = append(res, fmt.Sprintf("%v", p.Value))
		}
	}
	return res
}

func (t taskCmd) fullPath() (string, error) {
	p := t.path
	if !path.IsAbs(p) {
		var err error
		if p, err = exec.LookPath(p); err != nil {
			if t.l != nil {
				t.l.Errorf("Lookup of %q failed: %v", p, err)
			}
			return "", err
		}
	}

	return p, nil
}

func (t taskCmd) Do(params Params) (Artifacts, error) {
	p, err := t.fullPath()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(p, t.convertParams(params)...)
	if t.l != nil {
		t.l.Debugf("Executing command: %s", cmd)
	}

	var out []byte
	switch {
	case t.parser != nil:
		out, err = cmd.Output()
	default:
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	}
	if err != nil {
		if t.l != nil {
			t.l.Errorf("Command %s failed: %v", cmd, err)
		}
		return nil, err
	}

	if t.parser != nil {
		return t.parser(out)
	}

	return nil, nil
}
