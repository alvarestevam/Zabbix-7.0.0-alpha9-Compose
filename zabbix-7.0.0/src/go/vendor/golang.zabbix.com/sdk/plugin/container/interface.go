/*
** Zabbix
** Copyright 2001-2024 Zabbix SIA
**
** Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
** documentation files (the "Software"), to deal in the Software without restriction, including without limitation the
** rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
** permit persons to whom the Software is furnished to do so, subject to the following conditions:
**
** The above copyright notice and this permission notice shall be included in all copies or substantial portions
** of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
** WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
** COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
** TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
**/

package container

import "golang.zabbix.com/sdk/plugin"

type exportCtx struct {
	resultWriter emptyResultWriter
	matcher      emptyMatcher
	timeout      int
}

func (ctx exportCtx) ClientID() uint64 {
	return 0
}
func (ctx exportCtx) ItemID() uint64 {
	return 0
}
func (ctx exportCtx) Output() plugin.ResultWriter {
	return ctx.resultWriter
}
func (ctx exportCtx) Meta() *plugin.Meta {
	return nil
}
func (ctx exportCtx) GlobalRegexp() plugin.RegexpMatcher {
	return ctx.matcher
}
func (ctx exportCtx) Timeout() int {
	return ctx.timeout
}

func (ctx exportCtx) Delay() string {
	return ""
}

type emptyMatcher struct{}

func (em emptyMatcher) Match(value string, pattern string, mode int, output_template *string) (bool, string) {
	return false, ""
}

type emptyResultWriter struct{}

func (rw emptyResultWriter) Write(result *plugin.Result) {}
func (rw emptyResultWriter) Flush()                      {}
func (rw emptyResultWriter) SlotsAvailable() int         { return 0 }
func (rw emptyResultWriter) PersistSlotsAvailable() int  { return 0 }
