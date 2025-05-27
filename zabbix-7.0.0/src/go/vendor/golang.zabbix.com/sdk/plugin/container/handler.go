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

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
	"golang.zabbix.com/sdk/plugin/comms"
)

const (
	defaultTimeout = 3
	socketArg      = 2
	startTypeArg   = 3
)

const (
	Info    = 0
	Crit    = 1
	Err     = 2
	Warning = 3
	Debug   = 4
	Trace   = 5
)

// TODO: COMMENT
type handler struct {
	name          string
	accessor      plugin.Accessor
	socket        string
	registerStart bool
	connection    net.Conn
}

var agentProtocolVersion string

func NewHandler(name string) (handler, error) {
	h := handler{
		name: name,
	}

	if len(os.Args) < socketArg {
		return handler{}, errs.New("no socket provided")
	}

	h.socket = os.Args[1]

	if len(os.Args) < startTypeArg {
		h.registerStart = false

		return h, nil
	}

	var err error

	h.registerStart, err = strconv.ParseBool(os.Args[2])
	if err != nil {
		return handler{}, errs.Wrap(err, "failed to parse third parameter")
	}

	return h, nil
}

func (h *handler) Execute() error {
	err := h.setConnection(h.socket, defaultTimeout*time.Second)
	if err != nil {
		return errs.Wrap(err, "failed to set connection")
	}

	h.accessor, err = plugin.GetByName(h.name)
	if err != nil {
		h.Errf("failed to get accessor for plugin %s, %s", h.name, err.Error())

		return errs.Wrap(err, "failed to get accessor")
	}

	h.run()

	return nil
}

func (h *handler) run() {
	go ignoreSIGINTandSIGTERM()

	for {
		err := h.handle()
		if err != nil {
			h.Errf("failed to handle request for plugin %s, %s", h.name, err.Error())
		}
	}
}

func (h *handler) handle() error {
	reqType, data, err := comms.Read(h.connection)
	if err != nil {
		return errs.Wrap(err, "failed to read request data")
	}

	h.Tracef("plugin %s executing %s request", h.name, comms.GetRequestName(reqType))

	switch reqType {
	case comms.RegisterRequestType:
		err = h.register(data)
		if err != nil {
			return err
		}
	case comms.StartRequestType:
		err = h.start()
		if err != nil {
			return err
		}
	case comms.TerminateRequestType:
		h.terminate()
	case comms.ValidateRequestType:
		err = h.validate(data)
		if err != nil {
			return err
		}
	case comms.ExportRequestType:
		go func() {
			err := h.export(data) //nolint:govet
			if err != nil {
				h.Errf("failed to handle request for plugin %s, %s", h.name, err.Error())

				return
			}

			h.Tracef("plugin %s export request completed", h.name)
		}()
	case comms.ConfigureRequestType:
		err = h.configure(data)
		if err != nil {
			return err
		}
	default:
		return errs.Errorf("unknown request recivied: %d", reqType)
	}

	h.Tracef("plugin %s executed %s request", h.name, comms.GetRequestName(reqType))

	return nil
}

func (h *handler) start() error {
	p, ok := h.accessor.(plugin.Runner)
	if !ok {
		return nil
	}

	p.Start()

	return nil
}

func (h *handler) stop() {
	if h.registerStart {
		return
	}

	p, ok := h.accessor.(plugin.Runner)
	if !ok {
		return
	}

	p.Stop()
}

func (h *handler) register(data []byte) error {
	var req comms.RegisterRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		return err
	}

	response := createEmptyRegisterResponse(req.Id)

	err = checkVersion(req.ProtocolVersion)
	if err != nil {
		response.Error = err.Error()

		return comms.Write(h.connection, response)
	}

	var metrics []string

	for key, metric := range plugin.Metrics {
		metrics = append(metrics, key)
		metrics = append(metrics, metric.Description)
	}

	interfaces := h.getInterfaces()
	response.Name = h.name
	response.Metrics = metrics
	response.Interfaces = interfaces

	return comms.Write(h.connection, response)
}

func checkVersion(pluginProtocolVersion string) error {
	if agentProtocolVersion != pluginProtocolVersion {
		return fmt.Errorf(
			"plugin cannot be loaded by agent using protocol version %s, the supported version is %s",
			pluginProtocolVersion,
			agentProtocolVersion,
		)
	}

	return nil
}

func (h *handler) validate(data []byte) error {
	var req comms.ValidateRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		return err
	}

	response := createEmptyValidateResponse(req.Id)

	p, ok := h.accessor.(plugin.Configurator)
	if !ok {
		panic("plugin does not implement Configurator interface")
	}

	err = p.Validate(req.PrivateOptions)
	if err != nil {
		response.Error = err.Error()
	}

	return comms.Write(h.connection, response)
}

func (h *handler) configure(data []byte) error {
	var req comms.ConfigureRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		return err
	}

	p, ok := h.accessor.(plugin.Configurator)
	if !ok {
		panic("plugin does not implement Configurator interface")
	}

	p.Configure(req.GlobalOptions, req.PrivateOptions)

	return nil
}

func (h *handler) export(data []byte) error {
	var req comms.ExportRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		return err
	}

	p, ok := h.accessor.(plugin.Exporter)
	if !ok {
		panic("plugin does not implement Exporter interface")
	}

	response := createEmptyExportResponse(req.Id)
	response.Value, err = p.Export(req.Key, req.Params, &exportCtx{timeout: req.Timeout})
	if err != nil {
		response.Error = err.Error()
	}

	return comms.Write(h.connection, response)
}

func (h *handler) terminate() {
	h.stop()
	os.Exit(0)
}

func (h *handler) getInterfaces() uint32 {
	var interfaces uint32

	_, ok := h.accessor.(plugin.Exporter)
	if ok {
		interfaces |= comms.Exporter
	}

	_, ok = h.accessor.(plugin.Configurator)
	if ok {
		interfaces |= comms.Configurator
	}

	_, ok = h.accessor.(plugin.Runner)
	if ok {
		interfaces |= comms.Runner
	}

	return interfaces
}

func (h *handler) Tracef(format string, args ...interface{}) {
	h.sendLog(createLogRequest(Trace, fmt.Sprintf(format, args...)))
}

func (h *handler) Debugf(format string, args ...interface{}) {
	h.sendLog(createLogRequest(Debug, fmt.Sprintf(format, args...)))
}

func (h *handler) Warningf(format string, args ...interface{}) {
	h.sendLog(createLogRequest(Warning, fmt.Sprintf(format, args...)))
}

func (h *handler) Infof(format string, args ...interface{}) {
	h.sendLog(createLogRequest(Info, fmt.Sprintf(format, args...)))
}

func (h *handler) Errf(format string, args ...interface{}) {
	h.sendLog(createLogRequest(Err, fmt.Sprintf(format, args...)))
}

func (h *handler) Critf(format string, args ...interface{}) {
	h.sendLog(createLogRequest(Crit, fmt.Sprintf(format, args...)))
}

func createLogRequest(severity uint32, message string) comms.LogRequest {
	return comms.LogRequest{
		Common: comms.Common{
			Id:   comms.NonRequiredID,
			Type: comms.LogRequestType,
		},
		Severity: severity,
		Message:  message,
	}
}

func createEmptyRegisterResponse(id uint32) comms.RegisterResponse {
	return comms.RegisterResponse{
		Common: comms.Common{
			Id:   id,
			Type: comms.RegisterResponseType,
		},
	}
}

func createEmptyExportResponse(id uint32) comms.ExportResponse {
	return comms.ExportResponse{
		Common: comms.Common{Id: id, Type: comms.ExportResponseType},
	}
}

func createEmptyValidateResponse(id uint32) comms.ValidateResponse {
	return comms.ValidateResponse{
		Common: comms.Common{Id: id, Type: comms.ValidateResponseType},
	}
}

func (h *handler) sendLog(request comms.LogRequest) {
	err := comms.Write(h.connection, request)
	if err != nil {
		panic(fmt.Sprintf("failed to log message %s", err.Error()))
	}
}

func ignoreSIGINTandSIGTERM() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	for {
		<-sigs
	}
}

func init() {
	agentProtocolVersion = comms.ProtocolVersion
}
