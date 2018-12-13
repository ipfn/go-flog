/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package flog_test

import (
	"bytes"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

	"github.com/ipfn/go-flog/flog"
	"github.com/ipfn/go-flog/flog/mock"
)

func TestNew(t *testing.T) {
	logging, err := flog.New(flog.Config{})
	assert.NoError(t, err)
	assert.Equal(t, zapcore.InfoLevel, logging.DefaultLevel())
	assert.Empty(t, logging.Levels())

	_, err = flog.New(flog.Config{
		LogSpec: "::=borken=::",
	})
	assert.EqualError(t, err, "invalid logging specification '::=borken=::': bad segment '=borken='")
}

func TestLoggingReset(t *testing.T) {
	logging, err := flog.New(flog.Config{})
	assert.NoError(t, err)

	var tests = []struct {
		desc string
		flog.Config
		err            error
		expectedRegexp string
	}{
		{
			desc:           "implicit log spec",
			Config:         flog.Config{Format: "%{message}"},
			expectedRegexp: regexp.QuoteMeta("this is a warning message\n"),
		},
		{
			desc:           "simple debug config",
			Config:         flog.Config{LogSpec: "debug", Format: "%{message}"},
			expectedRegexp: regexp.QuoteMeta("this is a debug message\nthis is a warning message\n"),
		},
		{
			desc:           "module error config",
			Config:         flog.Config{LogSpec: "test-module=error:info", Format: "%{message}"},
			expectedRegexp: "^$",
		},
		{
			desc:           "json",
			Config:         flog.Config{LogSpec: "info", Format: "json"},
			expectedRegexp: `{"level":"warn","ts":\d+\.\d+,"name":"test-module","caller":"flog/logging_test.go:\d+","msg":"this is a warning message"}`,
		},
		{
			desc:   "bad log spec",
			Config: flog.Config{LogSpec: "::=borken=::", Format: "%{message}"},
			err:    errors.New("invalid logging specification '::=borken=::': bad segment '=borken='"),
		},
		{
			desc:   "bad format",
			Config: flog.Config{LogSpec: "info", Format: "%{color:bad}"},
			err:    errors.New("invalid color option: bad"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			tc.Config.Writer = buf

			logging.ResetLevels()
			err := logging.Apply(tc.Config)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
				return
			}
			assert.NoError(t, err)

			logger := logging.Logger("test-module")
			logger.Debug("this is a debug message")
			logger.Warn("this is a warning message")

			assert.Regexp(t, tc.expectedRegexp, buf.String())
		})
	}
}

//go:generate counterfeiter -o mock/write_syncer.go -fake-name WriteSyncer . writeSyncer
type writeSyncer interface{ zapcore.WriteSyncer }

func TestLoggingSetWriter(t *testing.T) {
	ws := &mock.WriteSyncer{}

	logging, err := flog.New(flog.Config{})
	assert.NoError(t, err)

	logging.SetWriter(ws)
	logging.Write([]byte("hello"))
	assert.Equal(t, 1, ws.WriteCallCount())
	assert.Equal(t, []byte("hello"), ws.WriteArgsForCall(0))

	err = logging.Sync()
	assert.NoError(t, err)

	ws.SyncReturns(errors.New("welp"))
	err = logging.Sync()
	assert.EqualError(t, err, "welp")
}
