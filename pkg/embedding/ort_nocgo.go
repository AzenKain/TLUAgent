//go:build !cgo

package embedding

import "errors"

// ortValue wraps an ONNX Runtime tensor value behind a cgo-free interface.
type ortValue interface {
	Destroy() error
	floatData() ([]float32, []int64, bool)
}

// ortSession wraps a dynamic ONNX Runtime session.
type ortSession interface {
	Run(inputs, outputs []ortValue) error
	Destroy() error
}

type ortIOInfo struct {
	Name string
}

var errCgoDisabled = errors.New("onnx runtime is unavailable because this binary was built with CGO_ENABLED=0; rebuild with cgo enabled")

func ortSetLibraryPath(string) {}

func ortInitializeEnvironment() error {
	return errCgoDisabled
}

func ortNewDynamicSession(string, []string, []string) (ortSession, error) {
	return nil, errCgoDisabled
}

func ortNewInt64Tensor([]int64, []int64) (ortValue, error) {
	return nil, errCgoDisabled
}

func ortGetIOInfo(string) ([]ortIOInfo, []ortIOInfo, error) {
	return nil, nil, errCgoDisabled
}

func requireCgoRuntime() error {
	return errCgoDisabled
}
