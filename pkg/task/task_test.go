package task

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_taskCmd_convertParams(t *testing.T) {
	cases := []struct {
		params Params
		exp    []string
	}{
		{
			exp: []string{},
		},
		{
			params: Params{
				Param{Name: "-a"},
				Param{Name: "-lh"},
			},
			exp: []string{"-a", "-lh"},
		},
		{
			params: Params{
				Param{Name: "-str", Value: "s"},
				Param{Name: "-int", Value: 1},
				Param{Name: "-float", Value: 2.3},
				Param{Name: "out.flv"},
			},
			exp: []string{"-str", "s", "-int", "1", "-float", "2.3", "out.flv"},
		},
	}
	for _, c := range cases {
		var task taskCmd
		act := task.convertParams(c.params)
		assert.Equal(t, c.exp, act, "Failed on: %+v", c)
	}
}

func Test_taskCmd_fullPath(t *testing.T) {
	cases := []struct {
		task   taskCmd
		exp    string
		expErr error
	}{
		{
			expErr: fmt.Errorf("exec: \"\": executable file not found in $PATH"),
		},
		{
			task: taskCmd{
				path: "/abs/path/is/ok",
			},
			exp: "/abs/path/is/ok",
		},
		{
			task: taskCmd{
				path: "not_exist_command",
			},
			expErr: fmt.Errorf("exec: \"not_exist_command\": executable file not found in $PATH"),
		},
	}
	for _, c := range cases {
		act, actErr := c.task.fullPath()
		if c.expErr != nil {
			assert.Error(t, actErr, "No error on %+v", c)
			assert.Equal(t, c.expErr.Error(), actErr.Error(), "Failed on %+v", c)
			continue
		}
		assert.NoError(t, actErr, "Error on %+v", c)
		assert.Equal(t, c.exp, act, "Failed on %+v", c)
	}
}

func Test_taskCmd_Do(t *testing.T) {
	oldLevel := logrus.GetLevel()
	l := logrus.NewEntry(logrus.StandardLogger())
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(oldLevel)

	cases := []struct {
		task   taskCmd
		params Params
		exp    Artifacts
		expErr error
	}{
		// No params, no logger, no parser
		{
			task: taskCmd{
				path: "echo",
			},
		},
		// No params, have logger, no parser
		{
			task: taskCmd{
				path: "echo",
				l:    l,
			},
		},
		// Have params, logger, no parser
		{
			task: taskCmd{
				path: "echo",
				l:    l,
			},
			params: Params{
				Param{Name: "YYY"},
			},
		},
		// Have params, logger, parser
		{
			task: taskCmd{
				path: "echo",
				l:    l,
				parser: func(b []byte) (Artifacts, error) {
					return Artifacts{
						"output": string(b),
					}, nil
				},
			},
			params: Params{
				Param{Name: "XXX"},
			},
			exp: Artifacts{
				"output": "XXX\n",
			},
		},
		// Handling error without logger
		{
			task: taskCmd{
				path: "ls",
			},
			params: Params{
				Param{Name: "--unknown"},
			},
			expErr: fmt.Errorf("exit status 1"),
		},
		// Handling error with logger
		{
			task: taskCmd{
				path: "ls",
				l:    l,
			},
			params: Params{
				Param{Name: "--unknown"},
			},
			expErr: fmt.Errorf("exit status 1"),
		},
	}
	for _, c := range cases {
		act, actErr := c.task.Do(c.params)
		if c.expErr != nil {
			assert.Error(t, actErr, "No error on %+v", c)
			assert.Equal(t, c.expErr.Error(), actErr.Error(), "Failed on %+v", c)
			continue
		}
		assert.NoError(t, actErr, "Error on %+v", c)
		assert.Equal(t, c.exp, act, "Failed on %+v", c)

	}
}
